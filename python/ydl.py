import os
import subprocess
import sys
import tempfile

EXTRACTORS_PY = '''
from .youtube import (
    YoutubeIE,
    YoutubeChannelIE,
    YoutubeFavouritesIE,
    YoutubeHistoryIE,
    YoutubeLiveIE,
    YoutubePlaylistIE,
    YoutubePlaylistsIE,
    YoutubeRecommendedIE,
    YoutubeSearchDateIE,
    YoutubeSearchIE,
    YoutubeSearchURLIE,
    YoutubeShowIE,
    YoutubeSubscriptionsIE,
    YoutubeTruncatedIDIE,
    YoutubeTruncatedURLIE,
    YoutubeUserIE,
    YoutubeWatchLaterIE,
)

GenericIE = YoutubeIE
'''


def make_lite():
    extractor_dir = os.path.join('youtube_dl', 'extractor')

    r = subprocess.run([
        sys.executable,
        os.path.join('devscripts', 'make_lazy_extractors.py'),
        os.path.join(extractor_dir, 'lazy_extractors.py'),
    ])
    r.check_returncode()

    # noinspection PyUnresolvedReferences
    from youtube_dl.extractor import youtube

    files_required = set()
    files_required.add(os.path.join(extractor_dir, '__init__.py'))

    for mod in sys.modules:
        if not mod.startswith('youtube_dl.extractor.'):
            continue

        files_required.add(os.path.join(*mod.split('.')) + '.py')

    for fi in os.scandir(extractor_dir):
        if fi.is_file() and fi.path not in files_required:
            os.remove(fi.path)

    with open(os.path.join(extractor_dir, 'extractors.py'), 'w') as f:
        f.write(EXTRACTORS_PY)

    os.remove(os.path.join(extractor_dir, 'lazy_extractors.py'))

    r = subprocess.run([sys.executable, "ydl.py", "test"])
    r.check_returncode()


def test():
    import youtube_dl
    ydl = youtube_dl.YoutubeDL({
        "cachedir": tempfile.gettempdir(),
    })
    ydl.process_info = lambda a: None
    ydl.download(['https://www.youtube.com/watch?v=oHg5SJYRHA0'])


if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("command is required")
        sys.exit(1)

    cmd = sys.argv[1]
    if cmd == "make_lite":
        make_lite()
    elif cmd == "test":
        test()
    else:
        print("unknown command")
        sys.exit(1)
