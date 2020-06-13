#!/usr/bin/env python

import argparse
import contextlib
import copy
import datetime as dt
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


# ------------------------------------------------------------------------------

class YDLOpts:
    common = {
        'quiet': True,
        'noprogress': True,
        'youtube_include_dash_manifest': True,
        'no_color': True,
        'call_home': False,
        'ignoreerrors': False,
        'geo_bypass': True,
        'verbose': False,
        'prefer_ffmpeg': True,
    }
    download = {
        'write_all_thumbnails': True,
        'allsubtitles': True,
        'writesubtitles': True,
        'writeinfojson': True,
    }
    audio = {
        'format': 'bestaudio/best',
        'postprocessors': [{
            'key': 'FFmpegExtractAudio',
            'preferredcodec': 'mp3',
            'preferredquality': '64',
            'nopostoverwrites': True,
        }],
        'postprocessor_args': ['-ac', '1']
    }
    video = {
        'format': 'bestvideo+bestaudio/best',
        'merge_output_format': 'mkv',
    }


class Preset:
    def __init__(self, *, logger: typing.Optional[logging.Logger] = None):
        self.logger = logger

    @property
    def _download_opts(self) -> dict:
        opts = copy.deepcopy(YDLOpts.common)
        opts.update(YDLOpts.download)
        if self.logger:
            # noinspection PyTypeChecker
            opts['logger'] = self.logger
        return opts

    @property
    def info_opts(self) -> dict:
        opts = copy.deepcopy(YDLOpts.common)
        return opts

    @property
    def audio(self) -> dict:
        opts = self._download_opts
        opts.update(YDLOpts.audio)
        return opts

    @property
    def video(self) -> dict:
        opts = self._download_opts
        opts.update(YDLOpts.video)
        return opts


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


def get_logger(filename: typing.Optional[str] = None):
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
def sha256sum(filename: str) -> str:
    h = hashlib.sha256()
    b = bytearray(128 * 1024)
    mv = memoryview(b)
    with open(filename, 'rb', buffering=0) as f:
        for n in iter(lambda: f.readinto(mv), 0):
            h.update(mv[:n])
    return h.hexdigest()


# ------------------------------------------------------------------------------


def urls_hash(urls: typing.List[str]) -> str:
    h = hashlib.sha1()
    for url in sorted(urls):
        h.update(url.encode())
        h.update(b'::')
    return h.hexdigest()


class Download:
    def __init__(self, args: argparse.Namespace):
        self.urls = args.urls
        self.logger = get_logger(args.log)

        # ----------------------------------------------------------------------

        self.root = os.path.abspath(os.path.expanduser(args.root))

        tmp_dir = os.path.join(self.root, ".tmp")

        self.output_dir = os.path.join(tmp_dir, 'dl_{}'.format(urls_hash(args.urls)))
        os.makedirs(self.output_dir, exist_ok=True)

        # Cache for youtube-dl
        cache_dir = args.cache or os.path.join(tmp_dir, 'ydl_cache')
        os.makedirs(cache_dir, exist_ok=True)

        # ----------------------------------------------------------------------

        opts = getattr(Preset(logger=self.logger), args.preset)
        opts.update(
            outtmpl=os.path.join(self.output_dir, '%(upload_date)s_%(id)s/%(id)s.%(ext)s'),
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
            if not data.get('id') or data.get('is_live'):
                return
            infos[data['id']] = data
            return process_info(data)

        try:
            with mock.patch.object(ydl, 'process_info', process_hook):
                ydl.download(self.urls)
        except youtube_dl.DownloadError as exc:
            if exc.exc_info[0] in SYSTEM_EXCS:
                raise Error(str(exc), reason='system') from exc
            raise

        result = []

        for info in infos.values():

            dir_name = "{}_{}".format(info['upload_date'], info['id'])

            result_dir = os.path.join(self.output_dir, dir_name)
            if not os.path.exists(result_dir):
                continue

            upload_date = dt.datetime.strptime(info['upload_date'], '%Y%m%d').date()
            dest_dir = os.path.join(
                self.root,
                upload_date.strftime("%Y"),
                upload_date.strftime("%m"),
                dir_name,
            )

            os.makedirs(os.path.dirname(dest_dir), exist_ok=True)

            shutil.rmtree(dest_dir, ignore_errors=True)
            shutil.move(result_dir, dest_dir)

            files = []

            for path in glob.glob(os.path.join(dest_dir, "**"), recursive=True):
                self.logger.info("output file: %s", path)

                try:
                    fi = os.stat(path)
                except OSError as exc:
                    raise Error('could not stat output file') from exc

                if stat.S_ISDIR(fi.st_mode):
                    continue

                filehash = sha256sum(path)

                files.append({
                    "path": os.path.relpath(path, self.root),
                    "hash": filehash,
                    "size": fi.st_size,
                })

            redundant_keys = (
                'formats', 'requested_formats', 'format', 'format_id', 'requested_subtitles',
                *(k for k in info if str(k).startswith('_')),
            )
            for key in redundant_keys:
                info.pop(key, None)

            result.append({
                'id': info['id'],
                'files': files,
                'output_dir': self.output_dir,
                'info': info,
            })

        shutil.rmtree(self.output_dir, ignore_errors=True)

        return result


def arg_parser():
    parser = argparse.ArgumentParser()
    parser.add_argument('--log')

    subparsers = parser.add_subparsers(dest='command')

    subcmd = subparsers.add_parser('download')
    subcmd.set_defaults(func=Download)
    subcmd.add_argument('--root', required=True)
    subcmd.add_argument('--cache')
    subcmd.add_argument('--preset', choices=['video', 'audio'], default='video')
    subcmd.add_argument('urls', nargs='+')

    return parser


def main():
    args = arg_parser().parse_args()
    logger = get_logger(args.log)

    try:
        with suppress_output():
            result = args.func(args).execute()
        json_dump(result, sys.stdout)

    except Exception as exc:
        if isinstance(exc, Error):
            msg = str(exc)
            reason = exc.reason
        else:
            logger.exception("unknown error")
            msg = '{}: {}'.format(exc.__class__.__name__, str(exc))
            reason = 'unknown'

        json_dump({
            'error': msg,
            'reason': reason,
        }, sys.stderr)

        sys.exit(0xE7)


if __name__ == '__main__':
    main()
