# Librarian
Librarian package allows you to do 2 things:
- **Organize** your media in a deterministic way -- In particular, librarian/organizer would iterate over all the photos and videos in source directory, and generate a destination location based upon their EXIF timestamp (only for photos), and SHA256 checksum. Hence any duplicates would get caught, and can be deleted. This also has the main advantage that photos from multiple source directories can easily be merged in a consistent deterministic way.

- **Dedup** your media -- If you already have a directory full of photos and videos, you can run this tool to find both exact and approximate duplicates, given a matching threshold, and help you delete them. Approximate duplicates could mean files which match over 95%, and is very helpful for video files, which might have slightly different headers.

#### See it in action
[Organizing 70GB of media](http://showterm.io/20a33a98bc4fcfa0fe7d6) into a 52GB (de-duped) well arranged directory.

### Installation
```go
# Organizer
go get github.com/manishrjain/librarian/organize
go install github.com/manishrjain/librarian/organize

# Dedup
go get github.com/manishrjain/librarian/dedup
go install github.com/manishrjain/librarian/dedup
```

### Usage: Organize
```
$ organize
Usage of organize:
  -deletedups=false: Delete duplicates present in source folder.
  -dry=true: Don't commit the changes. Only show what would be performed
  -dst="": Choose root folder to move files to
  -numroutines=2: Number of routines to run.
  -src="": Choose directory to run this over
```

`organize --src dir1 --dst dir2`
This command would recursively iterate over all media files in dir1, and determine their final destination. This command would by default, only do a *dry run*. Any file moves would be shown, but not committed. Note that the destination directories might still get created even in dry run.

To actually run the file move commands:
`organize --src dir1 --dst dir2 --dry=false`

To also delete any exact duplicates:
`organize --src dir1 --dst dir2 --dry=false --deletedups=true`

###### Sample output
```
# organize --src Photos --dst ~/Pictures/Organized
DRY mode. No changes would be committed.
Using 2 routines
Found 77 files
Creating directory: /Users/mrjn/Pictures/Organized/2014Jun
Moving Photos/galaxy s3 dump/IMG_20140614_115916.jpg to /Users/mrjn/Pictures/Organized/2014Jun/14_1159_258bc291.jpeg
Moving Photos/20140616_195713.jpg to /Users/mrjn/Pictures/Organized/2014Jun/16_1957_8e4a5cfd.jpeg
Creating directory: /Users/mrjn/Pictures/Organized/2014Apr
Moving Photos/galaxy s3 dump/FB_IMG_13972886532495284.jpg to /Users/mrjn/Pictures/Organized/2014Apr/12_1744_f45d70a6.jpeg
Creating directory: /Users/mrjn/Pictures/Organized/Anarchs
Moving Photos/X-Men-Days-of-Future-Past-Cast-poster-570x829.jpg to /Users/mrjn/Pictures/Organized/Anarchs/f251eff990fcac35.jpeg
Moving Photos/20140606_141150.jpg to /Users/mrjn/Pictures/Organized/2014Jun/06_1411_d7399d53.jpeg
Moving Photos/Screenshot_2014-05-09-23-15-59.png to /Users/mrjn/Pictures/Organized/Anarchs/e7399ad131926b46.png
Moving Photos/Screenshot_2014-06-02-14-41-49.png to /Users/mrjn/Pictures/Organized/Anarchs/bf69c1f9b3e7dfb1.png
Moving Photos/Screenshot_2014-05-12-19-17-37.png to /Users/mrjn/Pictures/Organized/Anarchs/57897fff9eab3136.png
Moving Photos/Screenshot_2014-05-12-19-22-24.png to /Users/mrjn/Pictures/Organized/Anarchs/c24500ac26dc13e6.png
Moving Photos/Screenshot_2014-05-12-19-28-24.png to /Users/mrjn/Pictures/Organized/Anarchs/7a957001423153fa.png
Moving Photos/IMG_20140710_143501.jpg to /Users/mrjn/Pictures/Organized/Anarchs/9ef8155bde200bab.jpeg
Creating directory: /Users/mrjn/Pictures/Organized/2013Dec
Moving Photos/Ferrari-CascoRosso-23.jpg to /Users/mrjn/Pictures/Organized/2013Dec/09_0003_1382c74f.jpeg
...
```

### Usage: Dedup
```
$ dedup
Usage of dedup:
  -deletedups=false: Delete duplicate videos.
  -dir="": Choose directory to dedup video files.
  -percent=95: Video matching threshold ratio.
```

Dedup tool was written for video files, hence the terminology. But, it's a generic tool which can work on any other files too.

`dedup --dir dir1` would recursively iterate over all files in dir1, and if any two files have same size, run checksums over them to figure out how much do they match. No changes would be committed.

`dedup --dir dir1 --percent 99` would only show files which match over 99%.

`dedup --dir dir1 --deletedups=true` would also delete all the matching files, keeping one copy selected randomly. The matching threshold is set to 95% by default, and can be changed with `--percent` flag, as shown above.

###### Sample output
```
$ dedup --dir .
NO Match: 0.00 for [Flickr/2007.Oct/P1090722.JPG 1738493712.jpg] [Picasa/2012.Oct/DSC04311.JPG]
NO Match: 0.00 for [Flickr/2007.Sep/P1080865.JPG 1398910564.jpg] [Picasa/2009.Jun/DSC_0076.JPG]
Match: 100.00 for [Galaxy S3/Anarchs/IMG_20140611_223954.jpg] [Galaxy S3/Anarchs/IMG_20140611_224021.jpg]
Match: 100.00 for [Galaxy S3/Anarchs/Screenshot_2014-05-06-16-38-38.png] [Others/2014-05-06 16.38.39.png]
Match: 100.00 for [Galaxy S3/Anarchs/Screenshot_2014-05-06-16-38-56.png] [Others/2014-05-06 16.38.56.png]
Match: 100.00 for [Galaxy S3/Anarchs/Screenshot_2014-05-09-23-15-59.png] [Others/2014-05-09 23.16.00.png]
Match: 100.00 for [Galaxy S3/Anarchs/Screenshot_2014-05-09-23-16-20.png] [Others/2014-05-09 23.16.21.png]
Match: 100.00 for [Galaxy S3/Anarchs/Screenshot_2014-05-12-19-17-23.png] [Others/2014-05-12 19.17.24.png]
Match: 100.00 for [Galaxy S3/Anarchs/Screenshot_2014-05-12-19-17-37.png] [Others/2014-05-12 19.17.38.png]
Match: 100.00 for [Galaxy S3/Anarchs/Screenshot_2014-05-12-19-20-40.png] [Others/2014-05-12 19.20.40.png]
Match: 100.00 for [Galaxy S3/Anarchs/Screenshot_2014-05-12-19-22-24.png] [Others/2014-05-12 19.22.24.png]
Match: 100.00 for [Galaxy S3/Anarchs/Screenshot_2014-05-12-19-22-33.png] [Others/2014-05-12 19.22.34.png]
Match: 100.00 for [Galaxy S3/Anarchs/Screenshot_2014-05-12-19-23-24.png] [Others/2014-05-12 19.23.25.png]
Match: 100.00 for [Galaxy S3/Anarchs/Screenshot_2014-05-12-19-24-09.png] [Others/2014-05-12 19.24.09.png]
Match: 100.00 for [Galaxy S3/Anarchs/Screenshot_2014-05-12-19-28-24.png] [Others/2014-05-12 19.28.24.png]
Match: 100.00 for [Galaxy S3/Anarchs/Screenshot_2014-05-12-19-29-02.png] [Others/2014-05-12 19.29.03.png]
Match: 100.00 for [Galaxy S3/Anarchs/Screenshot_2014-05-12-19-29-29.png] [Others/2014-05-12 19.29.29.png]
```
