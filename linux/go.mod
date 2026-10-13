module tuigo-linux

go 1.23

require (
	github.com/BambooEngine/goibus v0.0.0-20261002210324-385291ea64c3
	github.com/godbus/dbus/v5 v5.1.0
	tuigo v0.0.0
)

replace tuigo => ../app
