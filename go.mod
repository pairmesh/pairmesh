module github.com/pairmesh/pairmesh

go 1.17

require (
	github.com/NYTimes/gziphandler v1.1.1
	github.com/atotto/clipboard v0.1.4
	github.com/coreos/go-semver v0.3.0
	github.com/denisbrodbeck/machineid v1.0.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/emersion/go-autostart v0.0.0-20210130080809-00ed301c8e9a
	github.com/fatih/color v1.13.0
	github.com/flynn/noise v1.0.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/jeremywohl/flatten v1.0.1
	github.com/ledisdb/ledisdb v0.0.0-20200510135210-d35789ec47e6
	github.com/libp2p/go-reuseport v0.1.0
	github.com/lxn/walk v0.0.0-20210112085537-c389da54e794
	github.com/pingcap/fn v0.0.0-20200306044125-d5540d389059
	github.com/pkg/errors v0.9.1
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	go.uber.org/atomic v1.7.0
	go.uber.org/zap v1.19.1
	golang.org/x/crypto v0.0.0-20211209193657-4570a0811e8b
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
	golang.org/x/sys v0.0.0-20211106132015-ebca88c72f68
	golang.org/x/tools v0.1.8
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	gorm.io/driver/mysql v1.2.1
	gorm.io/gorm v1.22.4
	inet.af/netaddr v0.0.0-20211027220019-c74959edd3b6
)

require (
	github.com/cupcake/rdb v0.0.0-20161107195141-43ba34106c76 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/edsrzf/mmap-go v0.0.0-20170320065105-0bce6a688712 // indirect
	github.com/go-sql-driver/mysql v1.6.0 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.0-20170215233205-553a64147049 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.3 // indirect
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e // indirect
	github.com/mattn/go-colorable v0.1.9 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/onsi/ginkgo v1.16.4 // indirect
	github.com/onsi/gomega v1.16.0 // indirect
	github.com/pelletier/go-toml v1.9.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/siddontang/go v0.0.0-20170517070808-cb568a3e5cc0 // indirect
	github.com/siddontang/rdb v0.0.0-20150307021120-fc89ed2e418d // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/syndtr/goleveldb v0.0.0-20160425020131-cfa635847112 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go4.org/intern v0.0.0-20211027215823-ae77deb06f29 // indirect
	go4.org/unsafe/assume-no-moving-gc v0.0.0-20211027215541-db492cf91b37 // indirect
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/net v0.0.0-20211112202133-69e39bad7dc2 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	gopkg.in/Knetic/govaluate.v3 v3.0.0 // indirect
)
replace (
	github.com/pairmesh/pairmesh/node/resources => ../github.com/pairmesh/pairmesh/node/resources
)