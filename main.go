package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

var IListFirstFile *os.File
var IListSecFile *os.File
var NameListFile *os.File

func get(id uint64, out *[]uint8) *error {
	NameListFile.Seek(int64(id*200), 0)
	NameListFile.Read((*out)[:])
	return nil
}

func set(name string) (uint64, *error) {
	buffer := make([]byte, 200)

	// increment name qnt
	NameListFile.Seek(0, 0)
	NameListFile.Read(buffer[:8])
	size := binary.LittleEndian.Uint64(buffer[:8])
	NameListFile.Seek(0, 0)
	binary.LittleEndian.PutUint64(buffer[:8], size+1)
	NameListFile.Write(buffer[:8])
	NameListFile.Seek(0, 2)

	// append the name to final of file
	copy(buffer[:], []byte(name))
	NameListFile.Write(buffer[:])

	return size, nil
}

func read(term string) ([]uint64, *error) {
	word := make([]byte, 200)
	buffer := make([]byte, 200)
	copy(word[:], []byte(term))
	IListFirstFile.Seek(0, 0)
	IListFirstFile.Read(buffer[:8])
	size := binary.LittleEndian.Uint64(buffer[:8])
	found := false
	adress := uint64(0)
	i := uint64(0)
	for i < size {
		// search the id in first file
		IListFirstFile.Read(buffer[:])
		if string(buffer) == string(word) {
			IListFirstFile.Read(buffer[:8])
			adress = binary.LittleEndian.Uint64(buffer[:8])
			found = true
			break
		} else {
			IListFirstFile.Read(buffer[:8])
		}
		i = i + 1
	}
	if found {
		// iterate over second list to find the id
		IListSecFile.Seek(int64(adress), 0)
		IListSecFile.Read(buffer[:1])
		lenth := uint64(buffer[0])
		ids := make([]uint64, lenth)
		j := uint64(0)
		k := 0
		for j < lenth {
			IListSecFile.Read(buffer[:8])
			ids[j] = binary.LittleEndian.Uint64(buffer[:8])
			j = j + 1
			k = k + 1
			if k >= 10 {
				IListSecFile.Read(buffer[:8])
				IListSecFile.Seek(int64(binary.LittleEndian.Uint64(buffer[:8]))+1, 0)
				k = 0
			}
		}
		return ids, nil
	}
	return nil, nil
}

func create(name string) *error {
	// get the new id
	id, _ := set(name)

	// iterate over each word
	i := 0
	for {
		// get the word
		buffer := ""
		for {
			if i >= len(name) || name[i] == byte(' ') {
				i = i + 1
				break
			}
			buffer = buffer + string(name[i])
			i = i + 1
		}
		lenth := len(buffer)
		word := make([]byte, 200)
		copy(word[:], []byte(buffer))

		// verify if the word is valid
		valid := true
		if lenth > 0 && lenth <= 200 {
			j := 0
			for j < lenth {
				if word[j] >= byte('A') && word[j] <= byte('Z') {
					word[j] = (word[j] - byte('A')) + byte('a')
				} else if !(word[j] >= byte('a') && word[j] <= byte('z')) {
					valid = false
					break
				}
				j = j + 1
			}
		} else {
			valid = false
		}

		// bind the id to word in list files
		if valid && string(word[:lenth]) != "de" && string(word[:lenth]) != "da" &&
			string(word[:lenth]) != "do" {
			fmt.Println(string(word))
			tmp := make([]byte, 200)
			IListFirstFile.Seek(0, 0)
			IListFirstFile.Read(tmp[:8])
			size := binary.LittleEndian.Uint64(tmp[:8])
			adress := uint64(0)
			pointer := uint64(0)
			found := false

			//fmt.Println(size)

			// try to find the adress that correspond of word in first file
			for !found && pointer < size {
				IListFirstFile.Read(tmp[:])
				fmt.Println(string(tmp))
				if string(tmp) == string(word) {
					IListFirstFile.Read(tmp[:8])
					adress = binary.LittleEndian.Uint64(tmp[:8])
					found = true
				} else {
					IListFirstFile.Read(tmp[:8])
				}
				pointer = pointer + 1
			}

			if found { // regist the id in the adress

				// increment the cell size
				IListSecFile.Seek(int64(adress), 0)
				IListSecFile.Read(tmp[:1])
				IListSecFile.Seek(int64(adress), 0)
				IListSecFile.Write([]byte{tmp[0] + 1})

				var addToSecFile func(id uint64, adress uint64, counter uint64) *error

				addToSecFile = func(id uint64, adress uint64, counter uint64) *error {
					if counter < 10 {
						IListSecFile.Seek(int64(counter*8+1)+int64(adress), 0)
						binary.LittleEndian.PutUint64(tmp[:8], id)
						IListSecFile.Write(tmp[:8])
					} else if counter == 10 {
						// write the new cell adress in final of actual cell in sec file
						secFileSize, _ := IListSecFile.Seek(0, 2)
						IListSecFile.Seek(int64(10*8+1)+int64(adress), 0)
						binary.LittleEndian.PutUint64(tmp[:8], uint64(secFileSize))
						IListSecFile.Write(tmp[:8])

						// write the id on new cell
						IListSecFile.Seek(0, 2)
						IListSecFile.Write([]byte{1})
						binary.LittleEndian.PutUint64(tmp[:8], id)
						IListSecFile.Write(tmp[:8])
					} else {
						// move to last cell in cell chain
						for counter >= 10 {
							IListSecFile.Seek(int64(10*8+1)+int64(adress), 0)
							IListSecFile.Read(tmp[:8])
							adress = binary.LittleEndian.Uint64(tmp[:8])
							counter = counter - 10
						}

						return addToSecFile(id, adress, counter)
					}
					return nil
				}

				return addToSecFile(id, adress, uint64(tmp[0]))
			} else {
				// if the word not found in first file, append in the final of
				// first file the word and the new adress(adress in sec file):
				// |----------------------------------------------------------|
				// |  word (200 bytes)          | adress in sec file (8 bytes)|
				// |----------------------------------------------------------|

				// increment first file size and move to end
				IListFirstFile.Seek(0, 0)
				binary.LittleEndian.PutUint64(tmp[:8], size+1)
				IListFirstFile.Write(tmp[:8])
				IListFirstFile.Seek(0, 2)

				// write word in first file
				IListFirstFile.Write(word[:])

				// increment the second file size
				IListSecFile.Seek(0, 0)
				IListSecFile.Read(tmp[:8])
				secFileSize := binary.LittleEndian.Uint64(tmp[:8])
				IListSecFile.Seek(0, 0)
				binary.LittleEndian.PutUint64(tmp[:8], secFileSize+1)
				IListSecFile.Write(tmp[:8])

				// get the adress, and write on first file
				adress = secFileSize*(1+8*10+8) + 8
				binary.LittleEndian.PutUint64(tmp[:8], adress)
				IListFirstFile.Write(tmp[:8])

				// regist the id on second file in new adress
				IListSecFile.Seek(int64(adress), 0)
				IListSecFile.Write([]byte{1})
				binary.LittleEndian.PutUint64(tmp[:8], id)
				IListSecFile.Write(tmp[:8])

				// erase the other id spaces and next pointer adress:
				// 8*9 (bytes) = 72 -> size of the other id spaces in
				//                     cell
				// 8   (bytes) -> size of next cell adress
				IListSecFile.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
				IListSecFile.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
				IListSecFile.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
				IListSecFile.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
				IListSecFile.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
				IListSecFile.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
				IListSecFile.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
				IListSecFile.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
				IListSecFile.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})
				IListSecFile.Write([]byte{255, 255, 255, 255, 255, 255, 255, 255})
			}
		}

		if i >= len(name) {
			break
		}
	}
	return nil
}

func main() {
	var err error

	NameListFile, err = os.OpenFile("db/name_list_file.db", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}
	NameListFile.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})

	IListFirstFile, err = os.OpenFile("db/ilist_first_file.db", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}
	IListFirstFile.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})

	IListSecFile, err = os.OpenFile("db/ilist_sec_file.db", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}
	IListSecFile.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})

	// some tests
	create("jose da silva Junqueira")
	create("Jose da Junqueira")
	create("jorge batatinha")
	create("Maria Jose da Silva")
	r, _ := read("jose")
	fmt.Println(r)
	r, _ = read("silva")
	fmt.Println(r)
	tmp := make([]byte, 200)
	_ = get(2, (&tmp))
	fmt.Println(string(tmp))
}
