package virtdev

import (
	"io/ioutil"
	"os"
	"testing"

	_ "github.com/mdzio/go-lib/testutil"
)

const expectedInterfaceList = `<?xml version="1.0" encoding="utf-8" ?> 
<interfaces v="1.0">
	<ipc>
	 	<name>BidCos-RF</name>
	 	<url>xmlrpc_bin://127.0.0.1:32001</url> 
	 	<info>BidCos-RF</info> 
	</ipc>
	<ipc>
	 	<name>VirtualDevices</name>
	 	<url>xmlrpc://127.0.0.1:39292/groups</url> 
	 	<info>Virtual Devices</info> 
	</ipc>
	<ipc>
	 	<name>HmIP-RF</name>
	 	<url>xmlrpc://127.0.0.1:32010</url>
	 	<info>HmIP-RF</info>
	</ipc>
	<ipc>
	 	<name>CCU-Jack</name>
	 	<url>xmlrpc://127.0.0.1:2121/RPC3</url>
	 	<info>CCU-Jack</info>
	</ipc>
</interfaces>
`

func TestAddToInterfaceList(t *testing.T) {
	err := addToInterfaceList(
		"testdata/InterfacesList.xml",
		"out.xml",
		"CCU-Jack",
		"xmlrpc://127.0.0.1:2121/RPC3",
		"CCU-Jack",
	)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("out.xml")

	content, err := ioutil.ReadFile("out.xml")
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != expectedInterfaceList {
		t.Fatalf("unexpected content: %s", string(content))
	}
}
