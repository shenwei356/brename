#!/bin/sh

dir=example

if [ -e $dir ]; then
    /bin/rm -rf $dir
fi

mkdir $dir

touch $dir/a.jpeg
touch $dir/a.html
touch $dir/b.jpeg

mkdir $dir/abc
touch $dir/abc/A.JPEG
touch $dir/abc/B.JPEG
touch $dir/abc/B.HTM

tree $dir
