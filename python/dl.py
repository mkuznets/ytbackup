#!/usr/bin/env python

import argparse
import contextlib
import copy
import datetime as dt
import glob
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

TEMP_ROOT = os.path.join(tempfile.gettempdir(), 'ytbackup')


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


def create_logger(filename):
    logger = logging.Logger("log")
    logger.setLevel(logging.DEBUG)
    f = open(filename, 'a+')

    handler = logging.StreamHandler(f)
    fmt = logging.Formatter('%(asctime)s\t%(levelname)s\t%(message)s')
    handler.setFormatter(fmt)

    logger.addHandler(handler)

    return logger


def create_hook(logger):
    def log_hook(data):
        logger.info(
            "%s, elapsed: %.1f, eta: %s",
            data.get('status', '<unknown status>'),
            data.get('elapsed', None),
            data.get('eta', None)
        )

    return log_hook


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
                'duration': data['duration'],
            })

        ydl = youtube_dl.YoutubeDL(Preset().info_opts)
        with mock.patch.object(ydl, 'process_info', process_info):
            ydl.download([self.url])

        return infos


class Download:
    def __init__(self, args: argparse.Namespace):
        if not shutil.which('zip'):
            raise Error('could not find zip binary')

        os.makedirs(TEMP_ROOT, exist_ok=True)
        self.output_dir = tempfile.mkdtemp(dir=TEMP_ROOT)

        self.url: str = args.url

        logger = None
        if args.log:
            logger = create_logger(args.log)

        opts: dict = getattr(Preset(logger=logger), args.preset)
        opts['outtmpl'] = os.path.join(
            self.output_dir, '%(upload_date)s_%(id)s/%(id)s.%(ext)s'
        )

        if logger:
            opts['progress_hooks'] = [create_hook(logger)]

        self.opts = opts

    def execute(self) -> typing.Any:
        import youtube_dl

        ydl = youtube_dl.YoutubeDL(self.opts)
        info = ydl.extract_info(self.url)

        os.chdir(self.output_dir)

        output_dirs = glob.glob('*')
        if not output_dirs:
            raise Error('could not find output dir')

        zip_path = f"{info['upload_date']}_{info['id']}.zip"

        r = sp.run(
            ['zip', '-q', '-0', '-r', zip_path, *output_dirs],
            stdout=sp.PIPE, stderr=sp.STDOUT
        )
        if r.returncode:
            raise Error(r.stdout.decode().strip())

        for d in output_dirs:
            shutil.rmtree(d, ignore_errors=True)

        upload_date = dt.datetime.strptime(info['upload_date'], '%Y%m%d').date()

        # Remove redundant keys
        for key in ('formats', 'requested_formats', 'format', 'format_id'):
            info.pop(key, None)

        return {
            'id': info['id'],
            'title': info.get('title', '<missing title>'),
            'uploader': info.get('uploader', '<unknown uploader>'),
            'upload_date': upload_date.isoformat(),
            'file': os.path.join(self.output_dir, zip_path),
            'info': info,
        }


def arg_parser():
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest='command', required=True)

    subcmd = subparsers.add_parser('info')
    subcmd.set_defaults(func=Info)
    subcmd.add_argument('url')

    subcmd = subparsers.add_parser('download')
    subcmd.set_defaults(func=Download)
    subcmd.add_argument('--log')
    subcmd.add_argument('--preset', choices=['video', 'audio'], default='video')
    subcmd.add_argument('url')

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
