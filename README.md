# FetchPKG

Fetch Sony Playstation 4 or 5 package updates referenced by a JSON-formatted manifest at the given URL

Ported [python script](https://gist.github.com/john-tornblom/5784dd2fb1c5267f2973f2e26842328c) to GO

## Example usage

Clone repo

```bash
$ go build main.go
$ main.exe -o ./Elden.pkg http://gs2.ww.prod.dl.playstation.net/gs2/ppkgo/prod/CUSA28863_00/33/f_68a4a386924400e31698cc0b44589dabeb689c42d789a183eb6575b1b1760ef1/f/UP0700-CUSA28863_00-ELDENRING0000000-A0118-V0100.json
```
or

```bash
$ go run main.go -o ./Elden.pkg http://gs2.ww.prod.dl.playstation.net/gs2/ppkgo/prod/CUSA28863_00/33/f_68a4a386924400e31698cc0b44589dabeb689c42d789a183eb6575b1b1760ef1/f/UP0700-CUSA28863_00-ELDENRING0000000-A0118-V0100.json
```
