package moduls

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	gopath "path"
)

type node struct {
	name     string
	offset   int64
	hash     []byte
	children []node
}

// TODO directory/file ==> merkel tree
func merkelify(path string) (root node) {
	info, err := os.Stat(path)
	HandlePanicError(err, "os.stat error, merkelify")

	if info.IsDir() {
		return hashDir(path)
	} else {
		return hashFile(path)
	}
}

// PS: this only includes the first 16 items in the directory
func hashDir(path string) node {
	var child node
	dir, err := os.ReadDir(path)
	HandlePanicError(err, "os.readdir err, hashDir")

	// if empty directory just hash the path and return
	if len(dir) == 0 {
		cHash := sha256.New()
		cHash.Write([]byte(path))
		child = node{
			name: gopath.Base(path),
			hash: cHash.Sum(nil),
		}
	}

	for i, de := range dir {
		filePath := path + "/" + de.Name()
		if de.IsDir() {
			child.children = append(child.children, hashDir(filePath))
		} else {
			child.children = append(child.children, hashFile(filePath))
		}

		// break after 16 items
		if i == 15 {
			break
		}
	}
	child.hash = calculateNodeHash(child.children)
	return child
}

func hashFile(path string) node {

	// child to be returned
	var child node

	file, err := os.Open(path)
	HandleFatalError(err, "error opening file "+path)
	defer file.Close()

	// making the hashes
	chunk := make([]byte, CHUNK_SIZE)
	var nodes []node

	// hash entire file and create nodes
	var i int64
	for {
		node := node{
			name:     fmt.Sprintf("/leaf%d", i),
			children: nil,
			offset:   i * CHUNK_SIZE,
		}

		n, err := file.Read(chunk)
		if err == io.EOF {
			break
		}
		HandleFatalError(err, "error reading "+path)

		tempHash := sha256.New()
		tempHash.Write(chunk[:n])
		node.hash = tempHash.Sum(nil)
		nodes = append(nodes, node)
	}

	child = makeBTree(nodes)
	child.name = gopath.Base(path)
	return child
}

// we are required to provide sources on code so for this one i did ask a friend for some help here (Mr. Scruff), just hints, not actual code
func makeBTree(sortedNodes []node) node {
	if len(sortedNodes) == 0 {
		return node{}
	}

	// leaf node
	if len(sortedNodes) <= MAX_CHILDREN {
		return node{children: sortedNodes}
	}

	// internal node
	var internalNode node
	internalNode.name = "/InternalNode"
	internalNode.hash = calculateNodeHash(sortedNodes)

	// the max children - 1 can just be done with a +1 outside but complicating things is fun
	perNode := (len(sortedNodes) + MAX_CHILDREN - 1) / MAX_CHILDREN

	for start := 0; start < len(sortedNodes); start += perNode {
		end := (start + 1) * perNode
		if end > len(sortedNodes) {
			end = len(sortedNodes)
		}
		child := makeBTree(sortedNodes[start:end])
		internalNode.children = append(internalNode.children, child)
	}
	internalNode.offset = internalNode.children[0].offset

	return internalNode
}

// to get the children's hash
func calculateNodeHash(nodes []node) []byte {
	hash := sha256.New()
	for _, n := range nodes {
		hash.Write(n.hash)
	}
	return hash.Sum(nil)
}
