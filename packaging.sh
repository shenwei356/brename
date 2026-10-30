#!/usr/bin/env sh

set -eu

name=brename
output_dir=binaries
targets="
linux/amd64
linux/arm64
linux/386
darwin/amd64
darwin/arm64
windows/amd64
windows/arm64
windows/386
"

# Validate the output before creating temporary files or replacing archives.
if [ -L "$output_dir" ] || [ "$output_dir" -ef . ] || { [ -e "$output_dir" ] && [ ! -d "$output_dir" ]; }; then
    echo "output directory must be a real directory: $output_dir" >&2
    exit 1
fi
mkdir -p "$output_dir"
staging_dir=$(mktemp -d "$output_dir/.$name-packaging.XXXXXX")
trap 'rm -rf "$staging_dir"' 0
trap 'exit 1' HUP INT TERM

# Finish every build and checksum before publishing any release file.
for target in $targets; do
    goos=${target%/*}
    goarch=${target#*/}
    extension=
    if [ "$goos" = windows ]; then
        extension=.exe
    fi

    binary="${name}${extension}"
    archive="${name}_${goos}_${goarch}${extension}.tar.gz"

    printf 'Building %s/%s\n' "$goos" "$goarch"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
        go build -trimpath -tags netgo \
        -ldflags "-extldflags '-static' -w -s" \
        -o "$staging_dir/$binary" .

    tar -C "$staging_dir" -zcf "$staging_dir/$archive" "$binary"
    (
        cd "$staging_dir"
        md5sum "$archive" > "$archive.md5.txt"
    )
    rm -f "$staging_dir/$binary"
done

# Check archives and checksum destinations before replacing the first file.
for file in "$staging_dir"/*.tar.gz "$staging_dir"/*.tar.gz.md5.txt; do
    destination=$output_dir/${file##*/}
    if [ -L "$destination" ] || { [ -e "$destination" ] && [ ! -f "$destination" ]; }; then
        echo "release destination must be a regular file: $destination" >&2
        exit 1
    fi
done
for file in "$staging_dir"/*.tar.gz "$staging_dir"/*.tar.gz.md5.txt; do
    mv -f "$file" "$output_dir/${file##*/}"
done
