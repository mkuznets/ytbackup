#!/usr/bin/env python

import argparse
import contextlib
import copy
import glob
import hashlib
import http.client
import json
import logging
import os
import shutil
import stat
import sys
import typing
import urllib.error
from unittest import mock

SYSTEM_EXCS = (urllib.error.URLError, http.client.HTTPException, OSError)

STDERR = sys.stderr

YDL_OPTIONS = {
    'buffersize': 16 * 1024,
    'quiet': True,
    'noprogress': True,
    'youtube_include_dash_manifest': True,
    'no_color': True,
    'call_home': False,
    'ignoreerrors': False,
    'geo_bypass': True,
    'verbose': False,
    'prefer_ffmpeg': True,
    'noplaylist': True,
    'write_all_thumbnails': True,
    'allsubtitles': True,
    'writesubtitles': True,
    'writeinfojson': True,
    'format': 'bestvideo+bestaudio/best',
    'merge_output_format': 'mkv',
}


# ------------------------------------------------------------------------------


class Error(Exception):

    def __init__(self, *args, reason=None, **kwargs):
        self.reason = reason or 'unknown'
        # noinspection PyArgumentList
        super().__init__(*args, **kwargs)


def json_dump(data, f: typing.TextIO):
    json.dump(
        data, f,
        indent=2,
        skipkeys=True,
        ensure_ascii=False,
        default=lambda x: None,
    )
    f.write('\n')


@contextlib.contextmanager
def suppress_output():
    with open(os.devnull, 'w') as f:
        with contextlib.redirect_stdout(f), contextlib.redirect_stderr(f):
            yield


def get_logger(filename: typing.Optional[str] = None) -> logging.Logger:
    logger = logging.getLogger("log")
    logger.setLevel(logging.DEBUG)

    if not logger.handlers:
        stream = STDERR
        if filename:
            stream = open(filename, 'a')

        handler = logging.StreamHandler(stream)

        fmt = logging.Formatter('%(asctime)s\t%(levelname)s\t%(message)s')
        handler.setFormatter(fmt)
        handler.setLevel(logging.DEBUG)

        logger.addHandler(handler)

    return logger


def create_progress_hook(logger):
    def log_hook(data):
        size_done = data.get('downloaded_bytes', None)
        size_total = data.get('total_bytes', None)

        report = {
            'finished': data.get('status') == 'finished',
            'done': 'unk',
        }

        if size_done is not None and size_total is not None:
            report['downloaded'] = size_done
            report['total'] = size_total
            report['done'] = '%.2f%%' % (size_done * 100 / size_total)

        logger.info("__progress__ %s", json.dumps(report))

    return log_hook


# noinspection PyUnresolvedReferences
def sha256sum(filename: str, logger: logging.Logger) -> str:
    h = hashlib.sha256()
    b = bytearray(128 * 1024)
    mv = memoryview(b)
    total = 0
    with open(filename, 'rb', buffering=0) as f:
        for i, n in enumerate(iter(lambda: f.readinto(mv), 0)):
            total += n
            if not (i % 160):
                logger.info("sha256: %d", total)
            h.update(mv[:n])
    return h.hexdigest()


# ------------------------------------------------------------------------------


class Download:
    def __init__(self, args: argparse.Namespace):
        self.url = args.url
        self.logger = get_logger(args.log)

        # ----------------------------------------------------------------------

        self.dest_dir = os.path.abspath(os.path.expanduser(args.dst))
        os.makedirs(os.path.dirname(self.dest_dir), exist_ok=True)

        self.root = os.path.abspath(os.path.expanduser(args.root))

        self.output_dir = tmp_dir = os.path.join(self.root, ".tmp")
        os.makedirs(self.output_dir, exist_ok=True)

        # Cache for youtube-dl
        cache_dir = args.cache or os.path.join(tmp_dir, 'ydl_cache')
        os.makedirs(cache_dir, exist_ok=True)

        # ----------------------------------------------------------------------

        opts = copy.copy(YDL_OPTIONS)
        opts.update(
            logger=self.logger,
            outtmpl=os.path.join(self.output_dir, '%(id)s/%(id)s.%(ext)s'),
            progress_hooks=[create_progress_hook(self.logger)],
            cachedir=cache_dir,
        )

        if args.log:
            ffmpeg_log = str(args.log).replace('.log', '-ffmpeg.log')
            opts['postprocessor_args'] = ['-progress', 'file:{}'.format(ffmpeg_log)]

        self.opts = opts

    def execute(self) -> typing.Any:
        import youtube_dl

        ydl = youtube_dl.YoutubeDL(self.opts)
        process_info = ydl.process_info

        infos = {}

        def process_hook(data):
            if not data.get('id'):
                return
            infos[data['id']] = data
            return process_info(data)

        try:
            with mock.patch.object(ydl, 'process_info', process_hook):
                ydl.download([self.url])
        except youtube_dl.DownloadError as exc:
            if exc.exc_info[0] in SYSTEM_EXCS:
                raise Error(str(exc), reason='system') from exc
            raise

        if not infos:
            raise Error("result is empty")

        result = []

        for info in infos.values():
            result_dir = os.path.join(self.output_dir, info['id'])
            if not os.path.exists(result_dir):
                raise Error("result directory is not found: %s".format(info['id']))

            shutil.rmtree(self.dest_dir, ignore_errors=True)
            shutil.move(result_dir, self.dest_dir)

            files = []

            for path in glob.glob(os.path.join(self.dest_dir, "**"), recursive=True):
                self.logger.info("output file: %s", path)
                try:
                    fi = os.stat(path)
                except OSError as exc:
                    raise Error('could not stat output file') from exc

                if stat.S_ISREG(fi.st_mode):
                    files.append({
                        "path": os.path.relpath(path, self.root),
                        "hash": sha256sum(path, self.logger),
                        "size": fi.st_size,
                    })

            result.append({
                'id': info['id'],
                'files': files,
            })

        return result


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--log')
    parser.add_argument('--root', required=True)
    parser.add_argument('--dst', required=True)
    parser.add_argument('--cache')
    parser.add_argument('url')

    args = parser.parse_args()
    logger = get_logger(args.log)

    try:
        with suppress_output():
            result = Download(args).execute()
        json_dump(result, sys.stdout)

    except Exception as exc:
        if isinstance(exc, Error):
            msg = str(exc)
            reason = exc.reason
        else:
            logger.exception("unknown error")
            msg = '{}: {}'.format(exc.__class__.__name__, str(exc))
            reason = 'unknown'

        json_dump({'error': msg, 'reason': reason}, sys.stderr)
        sys.exit(0xE7)


if __name__ == '__main__':
    main()
