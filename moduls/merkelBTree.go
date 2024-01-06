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
	Name     string
	NodeType int64
	Offset   int64
	Hash     []byte
	Children []Node
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
	child.NodeType = DIRECTORY
	dir, err := os.ReadDir(path)
	HandlePanicError(err, "os.readdir err, hashDir")

	// if empty directory just Hash the path and return
	if len(dir) == 0 {
		cHash := sha256.New()
		cHash.Write([]byte(path))
		child = Node{
			Name:     gopath.Base(path),
			Hash:     cHash.Sum(nil),
			NodeType: DIRECTORY,
		}
	}

	for i, de := range dir {
		filePath := path + "/" + de.Name()
		if de.IsDir() {
			child.Children = append(child.Children, hashDir(filePath))
		} else {
			child.Children = append(child.Children, hashFile(filePath))
		}

		// break after 16 items
		if i == 15 {
			break
		}
	}
	child.Hash = calculateNodeHash(child.Children)
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

	// Hash entire file and create nodes
	var i int64
	for {
		node := Node{
			Name:     fmt.Sprintf("%s/%d", path, i),
			Children: nil,
			Offset:   i * CHUNK_SIZE,
			NodeType: CHUNK,
		}

		n, err := file.Read(chunk)
		if err == io.EOF {
			break
		}
		HandleFatalError(err, "error reading "+path)

		tempHash := sha256.New()
		tempHash.Write(chunk[:n])
		node.Hash = tempHash.Sum(nil)
		nodes = append(nodes, node)
		i++
	}

	if len(nodes) < 2 {
		child.NodeType = CHUNK
	} else {
		child.NodeType = BIG_FILE
	}

	child = makeBTree(nodes)
	child.Name = gopath.Base(path)
	return child
}

// we are required to provide sources on code so for this one i did ask a friend for some help here (Mr. Scruff), just hints, not actual code

//Natalia: If file is bigger than 32KB this code fails!
//probably problem is in
// > for start := 0; start < len(sortedNodes); start += perNode {
func makeBTree(sortedNodes []Node) Node {
	if len(sortedNodes) == 0 {
		return Node{}
	}

	// leaf Node
	if len(sortedNodes) <= MAX_CHILDREN {
		return Node{Children: sortedNodes}
	}

	// internal Node
	var internalNode Node
	internalNode.Name = "/InternalNode"
	internalNode.Hash = calculateNodeHash(sortedNodes)
	internalNode.NodeType = BIG_FILE

	// the max Children - 1 can just be done with a +1 outside but complicating things is fun
	perNode := (len(sortedNodes) + MAX_CHILDREN - 1) / MAX_CHILDREN

	for start := 0; start < len(sortedNodes); start += perNode {
		end := (start + 1) * perNode
		if end > len(sortedNodes) {
			end = len(sortedNodes)
		}
		child := makeBTree(sortedNodes[start:end])
		internalNode.Children = append(internalNode.Children, child)
	}
	internalNode.Offset = internalNode.Children[0].Offset

	return internalNode
}

// to get the Children's Hash
func calculateNodeHash(nodes []Node) []byte {
	Hash := sha256.New()
	for _, n := range nodes {
		Hash.Write(n.Hash)
	}
	return Hash.Sum(nil)
}

// search for specific Hash in the tree
func getHash(root Node, targetHash []byte) (node *Node, value []byte) {

	if compareHash(root.Hash, targetHash) {
		path := strings.TrimSuffix(root.Name, "/")
		data := getDataWithOffset(path, root.Offset)
		return &root, data
	}

	for _, child := range root.Children {
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

// opens file at `path“ and returns the first 1024 bytes found at `Offset`
func getDataWithOffset(path string, Offset int64) []byte {
	file, err := os.Open(path)
	HandlePanicError(err, fmt.Sprintf("error opening file %s", path))

	defer file.Close()
	_, err = file.Seek(Offset, 0)
	HandlePanicError(err, fmt.Sprintf("error seeking to Offset %d in file %s", Offset, path))

	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	HandlePanicError(err, fmt.Sprintf("error reading from file %s @ Offset %d", path, Offset))

	return buffer[:n]
}

// print merkel
func PrintMerkelTree(root Node, indent string) {
	fmt.Printf("%s- %s (Type: %d)\n", indent, root.Name, root.NodeType)

	for _, child := range root.Children {
		PrintMerkelTree(child, indent+"  ")
	}
}
