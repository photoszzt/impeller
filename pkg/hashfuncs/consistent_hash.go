// This function is from https://github.com/zeromicro/go-zero/blob/master/core/hash/consistenthash.go
// MIT License
//
// Copyright (c) 2020 xiaoheiban_server_go
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package hashfuncs

import (
	"fmt"
	"sort"
	"strconv"

	"sharedlog-stream/pkg/utils"

	"github.com/spaolacci/murmur3"
)

const (
	TopWeight   = 100
	minReplicas = 100
	prime       = 16777619
)

type (
	// HashFunc defines the hash method.
	HashFunc func(data []byte) uint64

	// A ConsistentHash is a ring hash implementation.
	// not safe to share between goroutines
	ConsistentHash struct {
		hashFunc HashFunc
		ring     map[uint64][]interface{}
		nodes    map[string]struct{}
		keys     []uint64
		replicas int
	}
)

// Hash returns the hash value of data.
func Hash(data []byte) uint64 {
	return murmur3.Sum64(data)
}

// NewConsistentHash returns a ConsistentHash.
func NewConsistentHash() *ConsistentHash {
	return NewCustomConsistentHash(minReplicas, Hash)
}

// NewCustomConsistentHash returns a ConsistentHash with given replicas and hash func.
func NewCustomConsistentHash(replicas int, fn HashFunc) *ConsistentHash {
	if replicas < minReplicas {
		replicas = minReplicas
	}

	if fn == nil {
		fn = Hash
	}

	return &ConsistentHash{
		hashFunc: fn,
		replicas: replicas,
		ring:     make(map[uint64][]interface{}),
		nodes:    make(map[string]struct{}),
	}
}

// Add adds the node with the number of h.replicas,
// the later call will overwrite the replicas of the former calls.
func (h *ConsistentHash) Add(node interface{}) {
	h.AddWithReplicas(node, h.replicas)
}

// AddWithReplicas adds the node with the number of replicas,
// replicas will be truncated to h.replicas if it's larger than h.replicas,
// the later call will overwrite the replicas of the former calls.
func (h *ConsistentHash) AddWithReplicas(node interface{}, replicas int) {
	h.Remove(node)

	if replicas > h.replicas {
		replicas = h.replicas
	}

	nodeRepr := repr(node)
	h.addNode(nodeRepr)

	for i := 0; i < replicas; i++ {
		hash := h.hashFunc([]byte(nodeRepr + strconv.Itoa(i)))
		h.keys = append(h.keys, hash)
		h.ring[hash] = append(h.ring[hash], node)
	}

	sort.Slice(h.keys, func(i, j int) bool {
		return h.keys[i] < h.keys[j]
	})
}

// AddWithWeight adds the node with weight, the weight can be 1 to 100, indicates the percent,
// the later call will overwrite the replicas of the former calls.
func (h *ConsistentHash) AddWithWeight(node interface{}, weight int) {
	// don't need to make sure weight not larger than TopWeight,
	// because AddWithReplicas makes sure replicas cannot be larger than h.replicas
	replicas := h.replicas * weight / TopWeight
	h.AddWithReplicas(node, replicas)
}

// Get returns the corresponding node from h base on the given v.
func (h *ConsistentHash) Get(v interface{}) (interface{}, bool) {
	if len(h.ring) == 0 {
		return nil, false
	}

	hash := h.hashFunc([]byte(repr(v)))
	index := sort.Search(len(h.keys), func(i int) bool {
		return h.keys[i] >= hash
	}) % len(h.keys)

	nodes := h.ring[h.keys[index]]
	switch len(nodes) {
	case 0:
		return nil, false
	case 1:
		return nodes[0], true
	default:
		innerIndex := h.hashFunc([]byte(innerRepr(v)))
		pos := int(innerIndex % uint64(len(nodes)))
		return nodes[pos], true
	}
}

// Remove removes the given node from h.
func (h *ConsistentHash) Remove(node interface{}) {
	nodeRepr := repr(node)

	if !h.containsNode(nodeRepr) {
		return
	}

	for i := 0; i < h.replicas; i++ {
		hash := h.hashFunc([]byte(nodeRepr + strconv.Itoa(i)))
		index := sort.Search(len(h.keys), func(i int) bool {
			return h.keys[i] >= hash
		})
		if index < len(h.keys) && h.keys[index] == hash {
			h.keys = append(h.keys[:index], h.keys[index+1:]...)
		}
		h.removeRingNode(hash, nodeRepr)
	}

	h.removeNode(nodeRepr)
}

func (h *ConsistentHash) removeRingNode(hash uint64, nodeRepr string) {
	if nodes, ok := h.ring[hash]; ok {
		newNodes := nodes[:0]
		for _, x := range nodes {
			if repr(x) != nodeRepr {
				newNodes = append(newNodes, x)
			}
		}
		if len(newNodes) > 0 {
			h.ring[hash] = newNodes
		} else {
			delete(h.ring, hash)
		}
	}
}

func (h *ConsistentHash) addNode(nodeRepr string) {
	h.nodes[nodeRepr] = struct{}{}
}

func (h *ConsistentHash) containsNode(nodeRepr string) bool {
	_, ok := h.nodes[nodeRepr]
	return ok
}

func (h *ConsistentHash) removeNode(nodeRepr string) {
	delete(h.nodes, nodeRepr)
}

func innerRepr(node interface{}) string {
	return fmt.Sprintf("%d:%v", prime, node)
}

func repr(node interface{}) string {
	return utils.GetStringValue(node)
}
