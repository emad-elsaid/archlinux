//go:build !integration
// +build !integration

package news

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bradleyjkemp/cupaloy"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"

	"github.com/emad-elsaid/fest/yay/pkg/text"
)

const lastNews = `
<rss xmlns:atom="http://www.w3.org/2005/Atom" version="2.0">
   <channel>
      <title>Arch Linux: Recent news updates</title>
      <link>https://www.archlinux.org/news/</link>
      <description>The latest and greatest news from the Arch Linux distribution.</description>
      <atom:link href="https://www.archlinux.org/feeds/news/" rel="self" />
      <language>en-us</language>
      <lastBuildDate>Tue, 14 Apr 2020 16:30:32 +0000</lastBuildDate>
      <item>
         <title>zn_poly 0.9.2-2 update requires manual intervention</title>
         <link>https://www.archlinux.org/news/zn_poly-092-2-update-requires-manual-intervention/</link>
         <description>&lt;p&gt;The zn_poly package prior to version 0.9.2-2 was missing a soname link.</description>
         <dc:creator xmlns:dc="http://purl.org/dc/elements/1.1/">Antonio Rojas</dc:creator>
         <pubDate>Tue, 14 Apr 2020 16:30:30 +0000</pubDate>
         <guid isPermaLink="false">tag:www.archlinux.org,2020-04-14:/news/zn_poly-092-2-update-requires-manual-intervention/</guid>
      </item>
   </channel>
</rss>
`

const sampleNews = `<?xml version="1.0" encoding="utf-8"?>
[Omitted long matching line]
This has been fixed in 0.9.2-2, so the upgrade will need to overwrite the
untracked files created by ldconfig. If you get an error&lt;/p&gt;
&lt;pre&gt;&lt;code&gt;zn_poly: /usr/lib/libzn_poly-0.9.so  exists in filesystem
&lt;/code&gt;&lt;/pre&gt;
&lt;p&gt;when updating, use&lt;/p&gt;
&lt;pre&gt;&lt;code&gt;pacman -Syu --overwrite usr/lib/libzn_poly-0.9.so
&lt;/code&gt;&lt;/pre&gt;
[Omitted long matching line]
&lt;pre&gt;&lt;code&gt;nss: /usr/lib/p11-kit-trust.so exists in filesystem
lib32-nss: /usr/lib32/p11-kit-trust.so exists in filesystem
&lt;/code&gt;&lt;/pre&gt;
&lt;p&gt;when updating, use&lt;/p&gt;
&lt;pre&gt;&lt;code&gt;pacman -Syu --overwrite /usr/lib\*/p11-kit-trust.so
&lt;/code&gt;&lt;/pre&gt;
[Omitted long matching line]
python modules. This has been fixed in 3.20.3-2, so the upgrade will
need to overwrite the untracked pyc files that were created. If you get errors
such as these&lt;/p&gt;
&lt;pre&gt;&lt;code&gt;hplip: /usr/share/hplip/base/__pycache__/__init__.cpython-38.pyc exists in filesystem
hplip: /usr/share/hplip/base/__pycache__/avahi.cpython-38.pyc exists in filesystem
hplip: /usr/share/hplip/base/__pycache__/codes.cpython-38.pyc exists in filesystem
...many more...
&lt;/code&gt;&lt;/pre&gt;
&lt;p&gt;when updating, use&lt;/p&gt;
&lt;pre&gt;&lt;code&gt;pacman -Suy --overwrite /usr/share/hplip/\*
&lt;/code&gt;&lt;/pre&gt;
[Omitted long matching line]
&lt;pre&gt;&lt;code&gt;firewalld: /usr/lib/python3.8/site-packages/firewall/__pycache__/__init__.cpython-38.pyc exists in filesystem
firewalld: /usr/lib/python3.8/site-packages/firewall/__pycache__/client.cpython-38.pyc exists in filesystem
firewalld: /usr/lib/python3.8/site-packages/firewall/__pycache__/dbus_utils.cpython-38.pyc exists in filesystem
...many more...
&lt;/code&gt;&lt;/pre&gt;
&lt;p&gt;when updating, use&lt;/p&gt;
&lt;pre&gt;&lt;code&gt;pacman -Suy --overwrite /usr/lib/python3.8/site-packages/firewall/\*
&lt;/code&gt;&lt;/pre&gt;
[Omitted long matching line]
&lt;p&gt;Some of you may know me from the days when I was much more involved in Arch, but most of you probably just know me as a name on the website. I’ve been with Arch for some time, taking the leadership of this beast over from Judd back in 2007. But, as these things often go, my involvement has slid down to minimal levels over time. It’s high time that changes.&lt;/p&gt;
&lt;p&gt;Arch Linux needs involved leadership to make hard decisions and direct the project where it needs to go. And I am not in a position to do this.&lt;/p&gt;
&lt;p&gt;In a team effort, the Arch Linux staff devised a new process for determining future leaders. From now on, leaders will be elected by the staff for a term length of two years. Details of this new process can be found &lt;a href="https://wiki.archlinux.org/index.php/DeveloperWiki:Project_Leader"&gt;here&lt;/a&gt;&lt;/p&gt;
&lt;p&gt;In the first official vote with Levente Polyak (anthraxx), Gaetan Bisson (vesath), Giancarlo Razzolini (grazzolini), and Sven-Hendrik Haase (svenstaro) as candidates, and through 58 verified votes, a winner was chosen:&lt;/p&gt;
&lt;p&gt;&lt;strong&gt;Levente Polyak (anthraxx) will be taking over the reins of this ship. Congratulations!&lt;/strong&gt;&lt;/p&gt;
&lt;p&gt;&lt;em&gt;Thanks for everything over all these years,&lt;br /&gt;
[Omitted long matching line]
[Omitted long matching line]
with the old-style &lt;code&gt;--compress&lt;/code&gt; option up to version 3.1.0. Version 3.1.1 was
released on 2014-06-22 and is shipped by all major distributions now.&lt;/p&gt;
&lt;p&gt;So we decided to finally drop the bundled library and ship a package with
system &lt;code&gt;zlib&lt;/code&gt;. This also fixes security issues, actual ones and in future. Go
and blame those running old versions if you encounter errors with &lt;code&gt;rsync
[Omitted long matching line]
&lt;p&gt;zstd and xz trade blows in their compression ratio. Recompressing all packages to zstd with our options yields a total ~0.8% increase in package size on all of our packages combined, but the decompression time for all packages saw a ~1300% speedup.&lt;/p&gt;
&lt;p&gt;We already have more than 545 zstd-compressed packages in our repositories, and as packages get updated more will keep rolling in. We have not found any user-facing issues as of yet, so things appear to be working.&lt;/p&gt;
&lt;p&gt;As a packager, you will automatically start building .pkg.tar.zst packages if you are using the latest version of devtools (&amp;gt;= 20191227).&lt;br /&gt;
As an end-user no manual intervention is required, assuming that you have read and followed the news post &lt;a href="https://www.archlinux.org/news/required-update-to-recent-libarchive/"&gt;from late last year&lt;/a&gt;.&lt;/p&gt;
[Omitted long matching line]
intervention when you hit this message:&lt;/p&gt;
&lt;pre&gt;&lt;code&gt;:: installing xorgproto (2019.2-2) breaks dependency 'inputproto' required by lib32-libxi
:: installing xorgproto (2019.2-2) breaks dependency 'dmxproto' required by libdmx
:: installing xorgproto (2019.2-2) breaks dependency 'xf86dgaproto' required by libxxf86dga
:: installing xorgproto (2019.2-2) breaks dependency 'xf86miscproto' required by libxxf86misc
&lt;/code&gt;&lt;/pre&gt;
&lt;p&gt;when updating, use: &lt;code&gt;pacman -Rdd libdmx libxxf86dga libxxf86misc &amp;amp;&amp;amp; pacman -Syu&lt;/code&gt; to perform the upgrade.&lt;/p&gt;</description><dc:creator xmlns:dc="http://purl.org/dc/elements/1.1/">Andreas Radke</dc:creator><pubDate>Fri, 20 Dec 2019 13:37:40 +0000</pubDate><guid isPermaLink="false">tag:www.archlinux.org,2019-12-20:/news/xorg-cleanup-requires-manual-intervention/</guid></item></channel></rss>
`

func TestPrintNewsFeed(t *testing.T) {
	layout := "2006-01-02"
	str := "2020-04-13"
	lastNewsTime, _ := time.Parse(layout, str)

	type args struct {
		cutOffDate time.Time
		bottomUp   bool
		all        bool
		quiet      bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "all-verbose", args: args{bottomUp: true, cutOffDate: time.Now(), all: true, quiet: false}, wantErr: false},
		{name: "all-quiet", args: args{bottomUp: true, cutOffDate: lastNewsTime, all: true, quiet: true}, wantErr: false},
		{name: "latest-quiet", args: args{bottomUp: true, cutOffDate: lastNewsTime, all: false, quiet: true}, wantErr: false},
		{name: "latest-quiet-topdown", args: args{bottomUp: false, cutOffDate: lastNewsTime, all: false, quiet: true}, wantErr: false},
	}
	t.Setenv("TZ", "UTC")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gock.New("https://archlinux.org").
				Get("/feeds/news").
				Reply(200).
				BodyString(sampleNews)

			defer gock.Off()

			r, w, _ := os.Pipe()
			logger := text.NewLogger(w, w, strings.NewReader(""), false, "logger")

			err := PrintNewsFeed(context.Background(), &http.Client{}, logger,
				tt.args.cutOffDate, tt.args.bottomUp, tt.args.all, tt.args.quiet)
			assert.NoError(t, err)

			w.Close()
			out, _ := io.ReadAll(r)
			cupaloy.SnapshotT(t, out)
		})
	}
}

// GIVEN last build time at 13h00
// WHEN there's a news posted at 18h00
// THEN it should still be printed
func TestPrintNewsFeedSameDay(t *testing.T) {
	str := "2020-04-14T13:04:05Z"
	lastNewsTime, _ := time.Parse(time.RFC3339, str)

	gock.New("https://archlinux.org").
		Get("/feeds/news").
		Reply(200).
		BodyString(lastNews)

	defer gock.Off()

	r, w, _ := os.Pipe()
	logger := text.NewLogger(w, w, strings.NewReader(""), false, "logger")

	err := PrintNewsFeed(context.Background(), &http.Client{}, logger,
		lastNewsTime, true, false, false)
	assert.NoError(t, err)

	w.Close()
	out, _ := io.ReadAll(r)
	cupaloy.SnapshotT(t, out)
}
