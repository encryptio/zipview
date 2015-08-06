#!/bin/sh
exec uglifyjs \
    ext/zip.js \
    ext/inflate.js \
    ext/zip-ext.js \
    zipview.js \
    -m -c \
    > zipview.min.js
