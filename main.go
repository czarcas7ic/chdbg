// Copyright 2022 Cosmos Network Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cosmos/iavl"
	tmdb "github.com/tendermint/tm-db"
)

func main() {
	flag.Parse()

	if flag.NArg() != 3 {
		fmt.Fprintf(os.Stderr, "chdbg: exactly two directories must be specified\n")
		os.Exit(2)
	}
	if err := diff(flag.Arg(0), flag.Arg(1), flag.Arg(2)); err != nil {
		fmt.Fprintf(os.Stderr, "chdbg: %v\n", err)
		os.Exit(2)
	}
}

func diff(dbdir1, dbdir2, height string) error {
	var (
		t1 *iavl.MutableTree
		t2 *iavl.MutableTree
	)

	heightNum, err := strconv.Atoi(height)
	if err != nil {
		return err
	}

	for i, dir := range []string{dbdir1, dbdir2} {
		db, err := openDB(dir)
		if err != nil {
			return err
		}
		defer db.Close()
		// Match iavlviewer.
		const cacheSize = 10000
		tree, err := iavl.NewMutableTree(db, cacheSize, false)
		if err != nil {
			return fmt.Errorf("%s: %w", dir, err)
		}
		if _, err := tree.LoadVersion(int64(heightNum)); err != nil {
			return fmt.Errorf("%s: %w", dir, err)
		}
		if i == 0 {
			t1 = tree
		} else {
			t2 = tree
		}
	}

	imt1, err := t1.GetImmutable(int64(heightNum))
	if err != nil {
		return err
	}
	it1, err := imt1.Iterator(nil, nil, true)
	if err != nil {
		return err
	}
	imt2, err := t2.GetImmutable(int64(heightNum))
	if err != nil {
		return err
	}
	it2, err := imt2.Iterator(nil, nil, true)
	if err != nil {
		return err
	}
	h1, err := imt1.Hash()
	if err != nil {
		return err
	}
	h2, err := imt2.Hash()
	if err != nil {
		return err
	}
	diff := !bytes.Equal(h1, h2)
	if diff {
		fmt.Printf("chdbg: hash mismatch: %X != %X\n", h1, h2)
	}
	reports := 0
	reportf := func(format string, args ...any) {
		reports++
		const max = 10
		if reports > max {
			if reports == max+1 {
				fmt.Fprintf(os.Stderr, "chdgb: ... (additional diffs omitted)\n")
			}
			return
		}
		fmt.Fprintf(os.Stderr, "chdbg: "+format, args...)
	}
loop:
	for {
		switch {
		case it1.Valid() && it2.Valid():
			k1, v1 := it1.Key(), it1.Value()
			k2, v2 := it2.Key(), it2.Value()
			cmp := bytes.Compare(k1, k2)
			switch cmp {
			case -1:
				it1.Next()
				reportf("%s: missing key %s\n", dbdir2, parseWeaveKey(k1))
			case +1:
				it2.Next()
				reportf("%s: missing key %s\n", dbdir1, parseWeaveKey(k2))
			default:
				it1.Next()
				it2.Next()
				eq := bytes.Equal(v1, v2)
				if !eq {
					reportf("key %s: value mismatch\n", parseWeaveKey(k1))

					reportf("value a \n%s\n, value b \n%s\n\n\n", v1, v2)
				} else {
					// // This returns either membership or non-membership proof
					proof1, err := imt1.GetProof(k1)
					// if err != nil {
					// 	return err
					// }

					// ok, err := imt2.VerifyProof(proof1, k1)
					// if err != nil {
					// 	return err
					// }
					// if !ok {
					// 	reportf("key %s: key proofs differ\n", parseWeaveKey(k1))
					// }

					// The strategy below helps to print the exact values that got mismatched instead of log keys
					// in the whole branch where hashes differred.

					// If proof1 is membership proof -> return false
					// if proof1 is a non-membership proof and valid -> return true
					// if proof1 is a non-membership proof and failed verifying -> return true
					ok, err := imt2.VerifyNonMembership(proof1, k1)
					if err != nil {
						return err
					}
					if ok {
						reportf("proofs failed\n", parseWeaveKey(k1))
						reportf("value a \n%s\n, value b \n%s\n\n\n", v1, v2)
					}
				}
			}
		case it1.Valid():
			it1.Next()
			k1 := it1.Key()
			reportf("%s: missing key %s\n", dbdir2, parseWeaveKey(k1))
		case it2.Valid():
			it2.Next()
			k2 := it2.Key()
			reportf("%s: missing key %s\n", dbdir1, parseWeaveKey(k2))
		default:
			break loop
		}
	}
	it1.Close()
	it2.Close()
	if diff {
		return fmt.Errorf("database mismatch at version %d with %d differences", heightNum, reports)
	}
	return nil
}

// parseWeaveKey assumes a separating : where all in front should be ascii,
// and all afterwards may be ascii or binary
func parseWeaveKey(key []byte) string {
	cut := bytes.IndexRune(key, ':')
	if cut == -1 {
		return encodeID(key)
	}
	prefix := key[:cut]
	id := key[cut+1:]
	return fmt.Sprintf("%s:%s", encodeID(prefix), encodeID(id))
}

// casts to a string if it is printable ascii, hex-encodes otherwise
func encodeID(id []byte) string {
	for _, b := range id {
		if b < 0x20 || b >= 0x80 {
			return strings.ToUpper(hex.EncodeToString(id))
		}
	}
	return string(id)
}

func openDB(dir string) (tmdb.DB, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	if !strings.HasSuffix(dir, ".db") {
		return nil, fmt.Errorf("database directory %q must end with .db", dir)
	}
	dir = dir[:len(dir)-len(".db")]

	name := filepath.Base(dir)
	parent := filepath.Dir(dir)

	db, err := tmdb.NewGoLevelDB(name, parent)
	if err != nil {
		return nil, err
	}

	return tmdb.NewPrefixDB(db, []byte("s/k:lockup/")), nil
}
