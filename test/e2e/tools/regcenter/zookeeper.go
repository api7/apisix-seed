package regcenter

import (
	"e2e/tools/common"
	"fmt"
	"time"

	"github.com/go-zookeeper/zk"
)

type Zookeeper struct {
	conn   *zk.Conn
	prefix string
}

func NewZookeeper() *Zookeeper {
	conn, _, err := zk.Connect([]string{common.ZK_HOST}, time.Second*5)
	if err != nil {
		panic(err)
	}
	return &Zookeeper{
		conn:   conn,
		prefix: "/zookeeper",
	}
}

func (zookeeper *Zookeeper) Online(node *common.Node) error {
	nodeStr := `[{"host":"` + common.DOCKER_GATEWAY + `","port":` + node.Port + `}]`
	// zk does not allow duplicate registration
	path := zookeeper.prefix + "/" + node.ServiceName
	_, stat, err := zookeeper.conn.Exists(path)
	if stat != nil {
		return nil
	}
	_, err = zookeeper.conn.Create(path, []byte(nodeStr), 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		return err
	}

	fmt.Println("register instance to Zookeeper: ", node.String())
	return err
}

func (zookeeper *Zookeeper) Offline(node *common.Node) error {
	path := zookeeper.prefix + "/" + node.ServiceName
	_, stat, err := zookeeper.conn.Exists(path)
	if err != nil {
		return err
	}

	fmt.Println("offline instance to Zookeeper: ", node.String())
	return zookeeper.conn.Delete(path, stat.Version)

}

func (zookeeper *Zookeeper) Clean() error {
	fmt.Println("clean all service form zookeeper...")
	children, stat, err := zookeeper.conn.Children(zookeeper.prefix)
	if err != nil {
		return err
	}
	for _, p := range children {
		if p == "config" || p == "quota" {
			continue
		}
		if err := zookeeper.conn.Delete(zookeeper.prefix+"/"+p, stat.Version); err != nil {
			return err
		}
		fmt.Println("delete service, serviceName=", p)
	}
	return nil
}
