package transport

import "testing"

func TestParseMacOSUSBKindles(t *testing.T) {
	output := `
  | +-o Kindle Paperwhite@01100000  <class IOUSBHostDevice, id 0x10013fe4d, registered, matched, active, busy 0 (21 ms), retain 31>
  |       "idProduct" = 39297
  |       "iProduct" = 2
  |       "iSerialNumber" = 3
  |       "USB Product Name" = "Kindle Paperwhite"
  |       "kUSBSerialNumberString" = "GN433X0743220D4T"
  |       "USB Vendor Name" = "Amazon"
  |       "idVendor" = 6473
  |       "kUSBProductString" = "Kindle Paperwhite"
  |       "USB Serial Number" = "GN433X0743220D4T"
  |       "kUSBVendorString" = "Amazon"
`

	devices := parseMacOSUSBKindles(output)
	if len(devices) != 1 {
		t.Fatalf("expected one Kindle, got %#v", devices)
	}

	device := devices[0]
	if device.Name != "Kindle Paperwhite" {
		t.Fatalf("unexpected name: %#v", device)
	}

	if device.Serial != "GN433X0743220D4T" {
		t.Fatalf("unexpected serial: %#v", device)
	}

	if device.Backend != "mtp" || !device.Connected {
		t.Fatalf("unexpected device state: %#v", device)
	}
}
