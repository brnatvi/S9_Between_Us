package moduls

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	gopath "path"
	"strings"
)

type Node struct {
	name     string
	nodeType int64
	offset   int64
	hash     []byte
	children []Node
}

// TODO directory/file ==> merkel tree
func Merkelify(path string) (root Node) {
	info, err := os.Stat(path)
	HandlePanicError(err, "os.stat error, merkelify")

	var r Node

	if info.IsDir() {
		r = hashDir(path)
	} else {
		r = hashFile(path)
	}

	if LOG_PRINT_DATA {
		PrintMerkelTree(r, " ")
	}

	return r
}

// PS: this only includes the first 16 items in the directory
func hashDir(path string) Node {
	var child Node
	child.nodeType = DIRECTORY
	dir, err := os.ReadDir(path)
	HandlePanicError(err, "os.readdir err, hashDir")

	// if empty directory just hash the path and return
	if len(dir) == 0 {
		cHash := sha256.New()
		cHash.Write([]byte(path))
		child = Node{
			name:     gopath.Base(path),
			hash:     cHash.Sum(nil),
			nodeType: DIRECTORY,
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

func hashFile(path string) Node {

	// child to be returned
	var child Node

	file, err := os.Open(path)
	HandleFatalError(err, "error opening file "+path)
	defer file.Close()

	// making the hashes
	chunk := make([]byte, CHUNK_SIZE)
	var nodes []Node

	// hash entire file and create nodes
	var i int64
	for {
		node := Node{
			name:     fmt.Sprintf("%s/%d", path, i),
			children: nil,
			offset:   i * CHUNK_SIZE,
			nodeType: CHUNK,
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
		i++
	}

	if len(nodes) < 2 {
		child.nodeType = CHUNK
	} else {
		child.nodeType = BIG_FILE
	}

	child = makeBTree(nodes)
	child.name = gopath.Base(path)
	return child
}

// we are required to provide sources on code so for this one i did ask a friend for some help here (Mr. Scruff), just hints, not actual code
func makeBTree(sortedNodes []Node) Node {
	if len(sortedNodes) == 0 {
		return Node{}
	}

	// leaf Node
	if len(sortedNodes) <= MAX_CHILDREN {
		return Node{children: sortedNodes}
	}

	// internal Node
	var internalNode Node
	internalNode.name = "/InternalNode"
	internalNode.hash = calculateNodeHash(sortedNodes)
	internalNode.nodeType = BIG_FILE

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
func calculateNodeHash(nodes []Node) []byte {
	hash := sha256.New()
	for _, n := range nodes {
		hash.Write(n.hash)
	}
	return hash.Sum(nil)
}

// search for specific hash in the tree
func getHash(root Node, targetHash []byte) (node *Node, value []byte) {

	if compareHash(root.hash, targetHash) {
		path := strings.TrimSuffix(root.name, "/")
		data := getDataWithOffset(path, root.offset)
		return &root, data
	}

	for _, child := range root.children {
		result, data := getHash(child, targetHash)
		if result != nil {
			return result, data
		}
	}

	return nil, nil
}

func compareHash(hash1, hash2 []byte) bool {
	return fmt.Sprintf("%x", hash1) == fmt.Sprintf("%x", hash2)
}

// opens file at `path“ and returns the first 1024 bytes found at `offset`
func getDataWithOffset(path string, offset int64) []byte {
	file, err := os.Open(path)
	HandlePanicError(err, fmt.Sprintf("error opening file %s", path))

	defer file.Close()
	_, err = file.Seek(offset, 0)
	HandlePanicError(err, fmt.Sprintf("error seeking to offset %d in file %s", offset, path))

	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	HandlePanicError(err, fmt.Sprintf("error reading from file %s @ offset %d", path, offset))

	return buffer[:n]
}

// print merkel
func PrintMerkelTree(root Node, indent string) {
	fmt.Printf("%s- %s (Type: %d)\n", indent, root.name, root.nodeType)

	for _, child := range root.children {
		PrintMerkelTree(child, indent+"  ")
	}
}
