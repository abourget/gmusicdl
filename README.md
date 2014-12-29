Google Music Downloader
=======================

Rock your parties with Mixxx or some Other DJ software, but make sure
you can load songs from Google Music.

Setup your config in `gmusicdl.conf` (it's a JSON file), and run
`./gmusicdl`. It'll then listen on your clipboard (install `xclip` or
something, see https://github.com/atotto/clipboard ).

Also, install "id3v2" so you can write the tags with metadata obtained
from Google Music.

Go to Google Music, right-click a song (or use the vertical `...`
menu), select "Share" and "Get link".  Copy the link and `gmusicdl`
will catch the clipboard, and queue the download.

Enjoy!
