package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"github.com/teamnsrg/zcrypto/ct"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

func pushToFile(incoming <-chan *ct.LogEntry, wg *sync.WaitGroup, outputDirectory string, entriesPerFile int) {
	defer wg.Done()

	if _, err := ioutil.ReadDir(outputDirectory); err != nil {
		log.Fatal(err)
	}

	var currentFile *os.File
	var writer *csv.Writer
	var err error
	currentBin := -1
	for entry := range incoming {
		bin := int(entry.Index) / entriesPerFile
		if bin != currentBin {
			if currentFile != nil {
				writer.Flush()
				currentFile.Close()
			}
			filename := filepath.Join(outputDirectory, strconv.Itoa(bin*entriesPerFile)+"-"+strconv.Itoa((bin+1)*entriesPerFile)+".csv")
			currentFile, err = os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatal(err)
			}
			writer = csv.NewWriter(currentFile)
		}

		chainBytes := make([]byte, 0)
		chainB64 := make([]string, len(entry.Chain))
		for i, c := range entry.Chain {
			chainBytes = append(chainBytes, c...)
			chainB64[i] = base64.StdEncoding.EncodeToString(c)
		}

		chainHash := fmt.Sprintf("%x", sha256.Sum256(chainBytes))
		leafHash := fmt.Sprintf("%x", sha256.Sum256(entry.X509Cert.Raw))

		var leafB64 string
		if entry.Leaf.TimestampedEntry.EntryType == ct.X509LogEntryType {
			leafB64 = base64.StdEncoding.EncodeToString(entry.X509Cert.Raw)
		} else if entry.Leaf.TimestampedEntry.EntryType == ct.PrecertLogEntryType {
			leafB64 = base64.StdEncoding.EncodeToString(entry.Precert.Raw)
		}

		row := []string{
			entry.Server,
			strconv.FormatInt(entry.Index, 10),
			entry.Leaf.TimestampedEntry.EntryType.String(),
			leafHash,
			leafB64,
			chainHash,
			strings.Join(chainB64, "|"),
			strconv.FormatUint(entry.Leaf.TimestampedEntry.Timestamp, 10),
		}

		writer.Write(row)
	}

	if currentFile != nil {
		writer.Flush()
		currentFile.Close()
	}
}
