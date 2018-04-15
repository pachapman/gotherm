package main

/*
 * Attempting to read data from a thermocoupler using the MAX31855 Thermocouple to digital converter via SPI bus on a
 * Raspberry Pi 3 Model B+.  The datasheet can be found here: https://cdn-shop.adafruit.com/datasheets/MAX31855.pdf
 */

import (
	"encoding/binary"
	"fmt"
	"github.com/kidoman/embd"
	_ "github.com/kidoman/embd/host/all"
	"time"
)

type TempReading struct {
	Internal float32		// Internal reading
	Thermocouple float32	// Thermocouple reading
	Fault bool				// True if there is a fault
	FaultMessage string		// A message describing the fault, if any
	LastUpdate int64		// When the reading was updated
}

func main() {
	// Set up SPI
	if err := embd.InitSPI(); err != nil {
		fmt.Printf("Error initializing SPI")
		panic(err)
	}
	defer embd.CloseSPI()
	spiBus := embd.NewSPIBus(embd.SPIMode0, 0, 1000000, 8, 0)
	defer spiBus.Close()

	// Allocate buffer
	dataBuf := [4]uint8{0, 0, 0, 0}

	// Continually loop for readings
	for true {
		fmt.Println("time:", time.Now())

		// Read data from SPI
		if err := spiBus.TransferAndReceiveData(dataBuf[:]); err != nil {
			fmt.Printf("Error reading data from SPI: ", err)
		}

		// Parse the data
		reading := parseTempReading(dataBuf)

		// Output result
		fmt.Println("Raw data:", dataBuf)
		if reading.Fault {
			fmt.Println(reading.FaultMessage)
		} else {
			fmt.Printf("Internal Temp: %f C\tExternal Temp: %f C\n", reading.Internal, reading.Thermocouple)
		}

		// Pause before the next reading
		time.Sleep(500 * time.Millisecond)
	}
}

// Parses the data from the SPI and returns a TempReading object containing user-friendly information.
func parseTempReading(dataBuf [4]uint8) TempReading {
	reading := TempReading{0, 0, false, "", time.Now().Unix()}
	// Check the bits that report faults first
	checkErrors(&reading, dataBuf)
	if !reading.Fault {
		// No faults, Parse readings
		// My own code.  Internal Temp works, External Temp gives erroneously large values.
		parseInternalTemp(dataBuf, &reading)
		parseExternalTemp(dataBuf, &reading)

		// The below code was created with the Python example code for inspiration.
		// It works unless there are negative values.
		//i := binary.BigEndian.Uint32(dataBuf[:])
		//reading.External = parseExternalTemp(i)
		//reading.Internal = parseInternalTemp(i)
	}
	return reading
}

// Constants for bitmasks used to gather info from the data returned from SPI
// First byte
const ocErrBit byte =     128 // 10000000
const scgErrBit byte =     64 // 01000000
const scvErrBit byte =     32 // 00100000
const intTempSign byte =    8 // 00001000
const intTempByte1 byte =  15 // 00001111
// Second byte
const intTempByte2 byte = 255 // 11111111
// Third byte
const errorBit byte =     128 // 10000000
const tcTempByte1 byte =   63 // 00111111
const tcTempSign byte =    32 // 00100000
// Fourth byte
const tcTempByte2 byte =  255 // 11111111

// Check the bits that report faults
func checkErrors(reading *TempReading, data [4]uint8) {
	if data[2] &errorBit == 1 {
		reading.Fault = true
		if data[0] &ocErrBit != 0 {
			reading.FaultMessage = "Open circuit to thermometer probe"
		} else if data[0] &scgErrBit != 0 {
			reading.FaultMessage = "Thermometer probe shorted to ground"
		} else if data[0] &scvErrBit != 0 {
			reading.FaultMessage = "Thermometer probe shorted to power"
		}
	} else {
		reading.Fault = false
	}
}

// Parse the temp of the internal sensor
func parseInternalTemp(data [4]uint8, reading *TempReading) {
	// the internal temp is composed of last 4 bits of byte 1 and all of byte 2
	a := byte(data[0] & intTempByte1)
	b := byte(data[1])
	if (data[0] & intTempSign) != 0 {
		//TODO: Negative number, apply Two's Compliment
	}
	c := binary.BigEndian.Uint16([]byte{a, b})
	f := float32(c)
	reading.Internal = f * 0.0625
}

// Parse the temp of the thermocouple sensor
func parseExternalTemp(data [4]uint8, reading *TempReading) {
	// the external temp is composed of the last 6 bits of byte 3 and all of byte 4
	a := byte(data[2] & tcTempByte1)
	b := byte(data[3])
	if (data[0] & tcTempSign) != 0 {
		//TODO: Negative number, apply Two's Compliment
	}
	c := binary.BigEndian.Uint16([]byte{a, b})
	f := float32(c)
	reading.Thermocouple = f * 0.25
}

// code inspired by the python code... doesn't work for negative values
// works otherwise

//// Return the thermocouple temperature value in degrees celsius.
//func parseExternalTemp(v uint32) float32 {
//	// Ignore bottom 18 bits.  They are status info and internal temp
//	v >>= 18
//	if (v & 0x80000000) != 0 {
//		// Negative value, take 2's compliment
//		v -= 16384
//	}
//	// Scale by 0.25 degrees C per bit and return value.
//	f := float32(v)
//	return f * 0.25
//}
//
//// Return internal temperature value in degrees celsius.
//func parseInternalTemp(v uint32) float32 {
//	// Ignore bottom 4 bits of thermocouple data.
//	v >>= 4
//	// Grab bottom 11 bits as internal temperature data.
//	internal := v & 0x7FF
//	if (v & 0x800) != 0 {
//		// Negative value, take 2's compliment
//		internal -= 4096
//
//	}
//	// Scale by 0.0625 degrees C per bit and return value.
//	f := float32(internal)
//	return f * 0.0625
//}
