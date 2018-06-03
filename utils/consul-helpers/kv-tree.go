package consulhelpers

import (
	"errors"
	"mikrodock-cli/logger"

	consulAPI "github.com/hashicorp/consul/api"
)

type ConsulHelper struct {
	client *consulAPI.Client
}

type KVNode struct {
	Key    string
	Value  []byte
	Childs []*KVNode
}

func NewConsulHelper(client *consulAPI.Client) *ConsulHelper {
	return &ConsulHelper{
		client: client,
	}
}

func (ch *ConsulHelper) NewTree(rootKey string) *KVNode {
	return &KVNode{
		Key:    rootKey,
		Value:  []byte{},
		Childs: make([]*KVNode, 0),
	}
}

func (t *KVNode) AddChild(key string, value []byte) *KVNode {
	node := &KVNode{
		Key:   key,
		Value: value,
	}
	t.Childs = append(t.Childs, node)
	return node
}

func (t *KVNode) AddSubCategory(key string) *KVNode {
	node := &KVNode{
		Key:   key,
		Value: []byte{},
	}
	t.Childs = append(t.Childs, node)
	return node
}

func (t *KVNode) HasValue() bool {
	return len(t.Value) != 0
}

func (ch *ConsulHelper) SendTree(tree *KVNode) error {
	rootKey := tree.Key

	// First, we will try it sync before async lol :o
	for _, child := range tree.Childs {
		err := ch.walkTree(rootKey, child)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ch *ConsulHelper) walkTree(context string, tree *KVNode) error {
	if tree.HasValue() {
		// It's a leaf
		kv := &consulAPI.KVPair{
			Key:   context + "/" + tree.Key,
			Value: tree.Value,
		}
		_, err := ch.client.KV().Put(kv, nil)
		return err
	} else {
		// It's a subdirectory (empty or not)
		if len(tree.Childs) != 0 {
			// It's not empty : we doesn't send anything, just explore further
			for _, child := range tree.Childs {
				err := ch.walkTree(context+"/"+tree.Key, child)
				if err != nil {
					return err
				}
			}
		} else {
			// It's empty : register it with a trailing slash to close the path
			kv := &consulAPI.KVPair{
				Key:   context + "/" + tree.Key + "/",
				Value: tree.Value,
			}
			_, err := ch.client.KV().Put(kv, nil)
			return err
		}
	}
	return nil
}

func sendKV(client *consulAPI.Client, parentKey string, leaf *KVNode) error {
	if len(leaf.Childs) != 0 {
		return errors.New("Cannot create KV Pair : is not leaf")
	} else {
		logger.Debug("Consul.KVTree", "Sending key : "+parentKey+"/"+leaf.Key)
		_, err := client.KV().Put(&consulAPI.KVPair{
			Key:   parentKey,
			Value: leaf.Value,
		}, nil)

		return err
	}
}
