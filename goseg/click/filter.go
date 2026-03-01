package click

import (
	"groundseg/click/internal/response"
)

func filterResponse(resType string, pokeResponse string) (string, bool, error) {
	return response.ParsePokeResponse(resType, pokeResponse)
}

/*
func filterJamResponse(patp, jamType, response string) (string, noun.Noun, error) {
	responseSlice := strings.Split(response, "\n")
	for _, line := range responseSlice {
		// extract jammed noun
		if strings.Contains(line, "%avow") {
			var jam string
			// Find the index of "noun "
			index := strings.Index(line, "noun ")

			if index != -1 {
				// Slice the string from just after "noun " onward
				jam = line[index+len("noun "):]
			} else {
				return "", nil, fmt.Errorf("Unable to extract jam file from avow")
			}
			jam = strings.TrimPrefix(jam, "0x")

			// make noun
			jamAtom := new(big.Int)
			jamAtom.SetString(jam, 16)
			n := noun.Cue(jamAtom)

			// dump to file
			fileName := filepath.Join(
				"/opt/nativeplanet/groundseg/bak",
				fmt.Sprintf("%s-%s-%s.jam", patp, jamType, time.Now().Format("20060102-150405")),
			)
			binaryData := jamAtom.Bytes()

			// Create or open the file
			file, err := os.Create(fileName)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Error creating file:", err))
			}
			defer file.Close()

			// Write the binary data to the file
			_, err = file.Write(binaryData)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Error writing to file:", err))
			}
			return fileName, n, nil
		}
	}
	return "", nil, fmt.Errorf("Jam file thread failure")
}

*/
