## Rebble Store for pebble Backend/API
The Rebble Store is a Pebble Appstore replacement.
If you want to contribute join us on the [Pebble Dev Discord server](http://discord.gg/aRUAYFN), then head to `#appstore`.

## Requirements

Backend/API layer requires `git`, `go`, `npm`, and `apib2swagger`.

## Dev Environment Setup
Pull down the project within your `$GOPATH`'s src folder ($GOPATH is an
environment variable and is typically set to $HOME/go/ on \*nix). This can be
done by running (for example) the following set of commands:

		GOPATH=go/
		mkdir -p $GOPATH/src/pebble-dev
		git clone https://github.com/pebble-dev/rebblestore-api.git $GOPATH/src/pebble-dev/rebblestore-api

Please [go fmt your code](https://blog.golang.org/go-fmt-your-code) and run `go
test` before committing your changes. Some editor plugins (such as vim-go)
should be able to do this automatically before save.

## Build Process

### Backend
1. If you haven't already, you will need to run `go get -v .` within the
	 project directory.
2. Run either `make` to build everything, or `go build -v .` to just build the
	 go executable.

### Database
1. If you haven't already, download a copy of the Pebble App Store by using [this tool](https://github.com/azertyfun/PebbleAppStoreCrawler) (you can find a direct download on this page).
2. Either create a link, or move your PebbleAppStore folder to `$GOPATH/src/pebble-dev/rebblestore-api/PebbleAppStore`, such that the folder `$GOPATH/src/pebble-dev/rebblestore-api/apps` exists; for example: `ln -s ~/Documents/PebbleAppStore $GOPATH/src/pebble-dev/rebblestore-api/PebbleAppStore`.
3. Start `./rebblestore-api` and access http://localhost:8080/admin/rebuild/db to rebuild the database.
