package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestAllHelpers(t *testing.T) {
	Convey("Given isLocalHostname and markHostnamesLocal logic", t, func() {
		markHostnamesLocal("localhost", "127.0.0.1", "testserver")

		So(isLocalHostname("foo"), ShouldEqual, true)
		So(isLocalHostname("foo.bar"), ShouldEqual, true)
		So(isLocalHostname("testserver"), ShouldEqual, true)
		So(isLocalHostname("localhost"), ShouldEqual, true)
		So(isLocalHostname("127.0.0.1"), ShouldEqual, true)

		So(isLocalHostname("example.com"), ShouldEqual, false)
		markHostnamesLocal("example.com")
		So(isLocalHostname("example.com"), ShouldEqual, true)
	})
	Convey("Given justHostname logic", t, func() {
		So(justHostname("example.com"), ShouldEqual, "example.com")
		So(justHostname("example.com:8080"), ShouldEqual, "example.com")
	})
}
