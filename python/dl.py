#!/usr/bin/env python

import argparse
import contextlib
import copy
import datetime as dt
import hashlib
import http.client
import json
import logging
import os
import shutil
import socket
import subprocess as sp
import sys
import typing
import urllib.error
from unittest import mock

NETWORK_EXCS = (urllib.error.URLError, http.client.HTTPException, socket.error)
SUMS_FILENAME = "SHA256SUMS"

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
        'verbose': True,
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
            stream = open(filename, 'a+')

        handler = logging.StreamHandler(stream)

        fmt = logging.Formatter('%(asctime)s\t%(levelname)s\t%(message)s')
        handler.setFormatter(fmt)
        handler.setLevel(logging.DEBUG)

        logger.addHandler(handler)

    return logger


def create_progress_hook(logger):
    def log_hook(data):
        logger.info(
            "%s, elapsed: %.1f, eta: %s",
            data.get('status', '<unknown status>'),
            data.get('elapsed', None),
            data.get('eta', None)
        )

    return log_hook


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
        if not shutil.which('zip'):
            raise Error('could not find zip binary')

        self.urls: typing.List[str] = args.urls
        self.logger = get_logger(args.log)

        # ----------------------------------------------------------------------

        self.root: str = os.path.abspath(os.path.expanduser(args.root))

        tmp_dir: str = os.path.join(self.root, ".tmp")

        self.output_dir = os.path.join(tmp_dir, f'dl_{urls_hash(args.urls)}')
        os.makedirs(self.output_dir, exist_ok=True)

        # Cache for youtube-dl
        cache_dir = args.cache or os.path.join(tmp_dir, 'ydl_cache')
        os.makedirs(cache_dir, exist_ok=True)

        # ----------------------------------------------------------------------

        opts: dict = getattr(Preset(logger=self.logger), args.preset)
        opts.update(
            outtmpl=os.path.join(self.output_dir, '%(upload_date)s_%(id)s/%(id)s.%(ext)s'),
            progress_hooks=[create_progress_hook(self.logger)],
            cachedir=cache_dir,
        )

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
            if exc.exc_info[0] in NETWORK_EXCS:
                raise Error('network unavailable', reason='network') from exc
            raise

        os.chdir(self.output_dir)
        result = []

        for info in infos.values():
            video_dir = f"{info['upload_date']}_{info['id']}"
            if not os.path.exists(video_dir):
                continue

            upload_date = dt.datetime.strptime(info['upload_date'], '%Y%m%d').date()
            dest_dir = os.path.join(self.root, upload_date.strftime("%Y"), upload_date.strftime("%m"))
            os.makedirs(dest_dir, exist_ok=True)

            zip_path = os.path.join(dest_dir, video_dir + '.zip')

            try:
                os.remove(zip_path)
            except OSError:
                pass

            r = sp.run(
                ['zip', '-q', '-0', '-r', zip_path, video_dir],
                stdout=sp.PIPE, stderr=sp.STDOUT
            )
            if r.returncode:
                raise Error(r.stdout.decode().strip())

            try:
                fi = os.stat(zip_path)
            except OSError as exc:
                raise Error(f'could not stat zip file') from exc

            dest_path_rel = os.path.relpath(zip_path, self.root)

            filehash = sha256sum(zip_path)
            with open(os.path.join(self.root, SUMS_FILENAME), "a") as f:
                f.write(f"{filehash} *{dest_path_rel}\n")
                f.flush()
                os.fsync(f.fileno())

            redundant_keys = (
                'formats', 'requested_formats', 'format', 'format_id', 'requested_subtitles',
                *(k for k in info if str(k).startswith('_')),
            )
            for key in redundant_keys:
                info.pop(key, None)

            result.append({
                'id': info['id'],
                'file': zip_path,
                'filesize': fi.st_size,
                'filehash': filehash,
                'output_dir': self.output_dir,
                'storage_path': os.path.relpath(zip_path, self.root),
                'info': info,
            })

        return result


def arg_parser():
    parser = argparse.ArgumentParser()
    parser.add_argument('--log')

    subparsers = parser.add_subparsers(dest='command', required=True)

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
            msg = f'{exc.__class__.__name__}: {str(exc)}'
            reason = 'unknown'

        json_dump({
            'error': msg,
            'reason': reason,
        }, sys.stderr)

        sys.exit(0xE7)


if __name__ == '__main__':
    main()
