#!/usr/bin/env python

import argparse
import contextlib
import copy
import datetime as dt
import hashlib
import json
import logging
import os
import shutil
import subprocess as sp
import sys
import tempfile
import traceback
import typing
from unittest import mock

DEFAULT_TMP_ROOT = os.path.join(tempfile.gettempdir(), 'ytbackup')
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
    pass


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


def create_logger(filename: typing.Optional[str] = None):
    logger = logging.getLogger("log")
    logger.setLevel(logging.DEBUG)

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


def sha256sum(filename):
    h = hashlib.sha256()
    b = bytearray(128 * 1024)
    mv = memoryview(b)
    with open(filename, 'rb', buffering=0) as f:
        for n in iter(lambda: f.readinto(mv), 0):
            h.update(mv[:n])
    return f"sha256:{h.hexdigest()}"


# ------------------------------------------------------------------------------


class Info:
    def __init__(self, args: argparse.Namespace):
        self.url: str = args.url

    def execute(self) -> typing.Any:
        import youtube_dl

        infos = []

        def process_info(data):
            infos.append({
                'id': data['id'],
                'url': data['webpage_url'],
                'is_live': bool(data['is_live']),
                'duration': int(data['duration'] or 0),
            })

        ydl = youtube_dl.YoutubeDL(Preset().info_opts)
        with mock.patch.object(ydl, 'process_info', process_info):
            ydl.download([self.url])

        return infos


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

        tmp_root: str = os.path.abspath(args.tmp)

        self.output_dir = os.path.join(tmp_root, f'dl_{urls_hash(args.urls)}')
        os.makedirs(self.output_dir, exist_ok=True)

        self.urls: typing.List[str] = args.urls

        logger = create_logger(args.log or None)

        opts: dict = getattr(Preset(logger=logger), args.preset)
        opts['outtmpl'] = os.path.join(
            self.output_dir, '%(upload_date)s_%(id)s/%(id)s.%(ext)s'
        )
        opts['progress_hooks'] = [create_progress_hook(logger)]
        opts['cachedir'] = os.path.join(tmp_root, 'ydl_cache')

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

        with mock.patch.object(ydl, 'process_info', process_hook):
            ydl.download(self.urls)

        os.chdir(self.output_dir)

        result = []

        for info in infos.values():
            video_dir = f"{info['upload_date']}_{info['id']}"
            if not os.path.exists(video_dir):
                continue

            zip_filename = video_dir + '.zip'
            try:
                os.remove(zip_filename)
            except OSError:
                pass

            r = sp.run(
                ['zip', '-q', '-0', '-r', zip_filename, video_dir],
                stdout=sp.PIPE, stderr=sp.STDOUT
            )
            if r.returncode:
                raise Error(r.stdout.decode().strip())

            try:
                fi = os.stat(zip_filename)
            except OSError as exc:
                raise Error(f"could not stat zip file") from exc

            filesize = fi.st_size
            filehash = sha256sum(zip_filename)

            upload_date = dt.datetime.strptime(info['upload_date'], '%Y%m%d').date()

            redundant_keys = (
                'formats', 'requested_formats', 'format', 'format_id', 'requested_subtitles',
                *(k for k in info if str(k).startswith('_')),
            )
            for key in redundant_keys:
                info.pop(key, None)

            result.append({
                'id': info['id'],
                'title': info.get('title', '<missing title>'),
                'uploader': info.get('uploader', '<unknown uploader>'),
                'upload_date': upload_date.isoformat(),
                'file': os.path.join(self.output_dir, zip_filename),
                'filesize': filesize,
                'filehash': filehash,
                'info': info,
            })

        return result


def arg_parser():
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest='command', required=True)

    subcmd = subparsers.add_parser('info')
    subcmd.set_defaults(func=Info)
    subcmd.add_argument('url')

    subcmd = subparsers.add_parser('download')
    subcmd.set_defaults(func=Download)
    subcmd.add_argument('--tmp', default=DEFAULT_TMP_ROOT)
    subcmd.add_argument('--log')
    subcmd.add_argument('--preset', choices=['video', 'audio'], default='video')
    subcmd.add_argument('urls', nargs='+')

    return parser


def main():
    args = arg_parser().parse_args()
    try:
        with suppress_output():
            result = args.func(args).execute()
        json_dump(result, sys.stdout)

    except Exception as exc:
        json_dump({
            'error': f'{exc.__class__.__name__}: {str(exc)}',
            'traceback': traceback.format_exc(),
        }, sys.stderr)
        sys.exit(0xE7)


if __name__ == '__main__':
    main()
