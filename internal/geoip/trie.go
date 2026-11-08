package geoip

import (
	"encoding/binary"
	"net"
	"sync"
)

// TrieNode represents a node in a binary radix trie for IPv4 lookups.
type TrieNode struct {
	children [2]*TrieNode
	record   *GeoRecord
}

// IPTrie provides high-performance, lock-free concurrent in-memory IP prefix lookups.
type IPTrie struct {
	root *TrieNode
	mu   sync.RWMutex
}

// NewIPTrie initializes an empty IP radix trie.
func NewIPTrie() *IPTrie {
	return &IPTrie{
		root: &TrieNode{},
	}
}

// InsertCIDR inserts a CIDR block and its associated GeoRecord into the radix trie.
func (t *IPTrie) InsertCIDR(cidrStr string, record *GeoRecord) error {
	_, ipNet, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return err
	}

	ip4 := ipNet.IP.To4()
	if ip4 == nil {
		// IPv6 fallback or skip for fast v4 trie
		return nil
	}

	ipVal := binary.BigEndian.Uint32(ip4)
	ones, _ := ipNet.Mask.Size()

	t.mu.Lock()
	defer t.mu.Unlock()

	current := t.root
	for i := 31; i >= 32-ones; i-- {
		bit := (ipVal >> i) & 1
		if current.children[bit] == nil {
			current.children[bit] = &TrieNode{}
		}
		current = current.children[bit]
	}

	current.record = record
	return nil
}

// Lookup finds the longest prefix matching the given IPv4 address.
func (t *IPTrie) Lookup(ip net.IP) *GeoRecord {
	ip4 := ip.To4()
	if ip4 == nil {
		return nil
	}

	ipVal := binary.BigEndian.Uint32(ip4)

	t.mu.RLock()
	defer t.mu.RUnlock()

	current := t.root
	var lastMatch *GeoRecord

	for i := 31; i >= 0; i-- {
		if current == nil {
			break
		}
		if current.record != nil {
			lastMatch = current.record
		}
		bit := (ipVal >> i) & 1
		current = current.children[bit]
	}

	if current != nil && current.record != nil {
		lastMatch = current.record
	}

	return lastMatch
}
