package main

import "testing"

func TestDetectPlatform(t *testing.T) {
	cases := []struct {
		name string
		file string
		want Platform
	}{
		// The bug: a CI run id that happens to contain "386".
		{
			"run id containing 386 does not make an amd64 build 386",
			"terraform-provider-okd_linux_amd64_34947983860",
			Platform{"linux", "amd64"},
		},
		{
			"a version containing 386 does not either",
			"terraform-provider-foo_1.386.0_linux_amd64.zip",
			Platform{"linux", "amd64"},
		},
		{
			"a real 386 build is still 386",
			"terraform-provider-foo_1.2.3_linux_386.zip",
			Platform{"linux", "386"},
		},

		// The shapes that already worked have to keep working.
		{"plain amd64", "terraform-provider-foo_linux_amd64.zip", Platform{"linux", "amd64"}},
		{"darwin arm64", "terraform-provider-foo_1.2.3_darwin_arm64.tar.gz", Platform{"darwin", "arm64"}},
		{"windows", "terraform-provider-foo_1.2.3_windows_amd64.exe", Platform{"windows", "amd64"}},
		{"freebsd arm", "terraform-provider-foo_1.2.3_freebsd_arm.zip", Platform{"freebsd", "arm"}},
		{"dashes", "terraform-provider-foo-darwin-amd64", Platform{"darwin", "amd64"}},
		{"aarch64 spelling", "foo_linux_aarch64.tar.gz", Platform{"linux", "arm64"}},
		{"x86_64 spelling", "foo_linux_x86_64.tar.gz", Platform{"linux", "amd64"}},
		{"a path is ignored", "/tmp/3860/foo_linux_amd64", Platform{"linux", "amd64"}},

		// No separators: the substring fallback.
		{"no separators", "providerlinuxarm64", Platform{"linux", "arm64"}},
		{"nothing recognisable defaults", "provider", Platform{"linux", "amd64"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := detectPlatform(c.file); got != c.want {
				t.Errorf("detectPlatform(%q) = %+v, want %+v", c.file, got, c.want)
			}
		})
	}
}
