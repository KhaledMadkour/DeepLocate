package main

import (
	"strconv"
	"strings"
	"time"

	structure "dlocate/dataStructures"
	utils "dlocate/osutils"

	log "github.com/sirupsen/logrus"
)

// getPartitionFiles gets files of partition and its children
func getPartitionFiles(partitionIndex int, path string) []string {
	partition := indexInfo.getPartition(partitionIndex)

	// check that partition is related to the path
	if !partition.inSameDirection(path) {
		return []string{}
	}

	partitionFiles := partition.getPartitionFiles()

	fileNames := make([]string, partition.FilesNumber)
	i := 0
	for path, files := range partitionFiles {
		for _, fileName := range files {
			fileNames[i] = partition.Root + path + fileName
			i++
		}
	}
	for _, child := range partition.Children {
		fileNames = append(fileNames, getPartitionFiles(child, path)...)
	}

	return fileNames
}

// getPartitionClildren get the children of partioin that is related
// to the given path (either parent or child)
// and excludes the non relevant partitions
func getPartitionClildren(partitionIndex int, path string) []int {
	partition := indexInfo.getPartition(partitionIndex)

	// check that partition is related to the path
	if !partition.inSameDirection(path) {
		return []int{}
	}

	children := []int{partitionIndex}

	for _, child := range partition.Children {
		children = append(children, getPartitionClildren(child, path)...)
	}

	return children
}

// query: word to search
// path: directoy to search in
// searchContent : bool to indicate search content or not
func find(query, path string, searchContent bool) ([]string, []string) {

	partitionIndex := directoryPartition.getPathPartition(path)

	log.Info("Start searching file names ...")

	// get all files names in the partition and its children
	fileNames := getPartitionFiles(partitionIndex, path)

	var matchedFiles []string

	scores := make(map[string]int) // map from file path to its score

	// exact match has high score
	for _, fileName := range fileNames {
		lastslash := strings.LastIndex(fileName, "/")
		exactFileName := fileName[lastslash+1:]
		if exactFileName == query {
			scores[fileName] += 10
		}
	}

	// partial match
	words := strings.Fields(query)
	for _, word := range words {

		for _, fileName := range fileNames {
			if strings.Contains(fileName, word) {
				scores[fileName]++
			}
		}
	}

	// score maybe used later to sort of filter first n entries
	for fileName := range scores {
		log.WithFields(log.Fields{
			"fileName": fileName,
		}).Info("found this file matched")
		matchedFiles = append(matchedFiles, fileName)

	}

	var contentMatchedFiles []string

	if searchContent {
		log.Info("Start searching file content ...")

		invertedIndex.Load()
		children := getPartitionClildren(partitionIndex, path)
		contentMatchedFiles = invertedIndex.Search(children, query, -1)

		for _, fileName := range contentMatchedFiles {
			log.WithFields(log.Fields{
				"fileName": fileName,
			}).Info("found this file content matched")

		}

	}
	return matchedFiles, contentMatchedFiles
}

func metaSearch() []string {
	//Size - file size in bytes
	//CTime - change time (last file name or path change)
	//MTime - modify time Max(last content change, CTime)
	//ATime - access time Max(last opened, MTime)

	start := utils.FileMetadata{ATime: time.Date(2019, 1, 1, 20, 34, 58, 651387237, time.UTC)}
	end := utils.FileMetadata{}

	//get parition index:
	partitionIndex := 0
	val, ok := indexInfo.metaCache.Get(strconv.Itoa(partitionIndex))
	var tree structure.KDTree
	if !ok {
		tree = readPartitionMetaGob(partitionIndex)
		indexInfo.metaCache.Set(strconv.Itoa(partitionIndex), tree)
	} else {
		tree = val.(structure.KDTree)
	}
	filesInfo := tree.SearchPartial(&start, &end)

	var files []string

	for _, file := range filesInfo {
		files = append(files, file.Path)
	}

	return files
}
