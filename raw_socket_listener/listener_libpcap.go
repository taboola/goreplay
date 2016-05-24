// +build !windows

package rawSocket

import (
    "runtime"
    "github.com/google/gopacket"
    "github.com/google/gopacket/layers"
    "github.com/google/gopacket/pcap"
    "log"
    "io"
    "sync"
    "strconv"
    "encoding/binary"
)


// DeviceNotFoundError raised if user specified wrong ip
type DeviceNotFoundError struct {
    addr string
}

func (e *DeviceNotFoundError) Error() string {
    devices, _ := pcap.FindAllDevs()

    if len(devices) == 0 {
        return "Can't get list of network interfaces, ensure that you running Gor as root user or sudo.\nTo run as non-root users see this docs https://github.com/buger/gor/wiki/Running-as-non-root-user"
    }

    var msg string
    msg += "Can't find interfaces with addr: " + e.addr + ". Provide available IP for intercepting traffic: \n"
    for _, device := range devices {
        msg += "Name: " + device.Name + "\n"
        if device.Description != "" {
            msg += "Description: " + device.Description + "\n"
        }
        for _, address := range device.Addresses {
            msg += "- IP address: " + address.IP.String() + "\n"
        }
    }

    return msg
}

func findPcapDevices(addr string) (interfaces []pcap.Interface, err error) {
    devices, err := pcap.FindAllDevs()
    if err != nil {
        log.Fatal(err)
    }

    for _, device := range devices {
        if (addr == "" || addr == "0.0.0.0" || addr == "[::]" || addr == "::") && len(device.Addresses) > 0 {
            interfaces = append(interfaces, device)
            continue
        }

        for _, address := range device.Addresses {
            if device.Name == addr || address.IP.String() == addr {
                interfaces = append(interfaces, device)
                return interfaces, nil
            }
        }
    }

    if len(interfaces) == 0 {
        return nil, &DeviceNotFoundError{addr}
    } else {
        return interfaces, nil
    }
}

func (t *Listener) readPcap() {
    devices, err := findPcapDevices(t.addr)
    if err != nil {
        log.Fatal(err)
    }

    bpfSupported := true
    if runtime.GOOS == "darwin" {
        bpfSupported = false
    }

    var wg sync.WaitGroup
    wg.Add(len(devices))

    for _, d := range devices {
        go func(device pcap.Interface) {
            handle, err := pcap.OpenLive(device.Name, 65536, true, t.messageExpire)
            if err != nil {
                log.Println("Pcap Error while opening device", device.Name, err)
                wg.Done()
                return
            }
            defer handle.Close()

            t.mu.Lock()
            t.connHandles = append(t.connHandles, wrongCloserProxy{handle})

            if bpfSupported {
                var bpf string

                if t.trackResponse {
                    bpf = "tcp port " + strconv.Itoa(int(t.port))
                } else {
                    bpf = "tcp dst port " + strconv.Itoa(int(t.port))
                }
                if err := handle.SetBPFFilter(bpf); err != nil {
                    log.Println("BPF filter error:", err, "Device:", device.Name)
                    wg.Done()
                    return
                }
            }
            t.mu.Unlock()

            linkType := handle.LinkType()
            source := gopacket.NewPacketSource(handle, linkType)
            source.Lazy = true
            source.NoCopy = true

            wg.Done()

            var data, srcIP []byte

            for {
                packet, err := source.NextPacket()

                if err == io.EOF {
                    break
                } else if err != nil {
                    continue
                }

                if linkType == layers.LinkTypeEthernet {
                    // Skip ethernet layer, 14 bytes
                    data = packet.Data()[14:]
                } else if linkType == layers.LinkTypeNull || linkType == layers.LinkTypeLoop {
                    data = packet.Data()[4:]
                }

                version := uint8(data[0]) >> 4

                if version == 4 {
                    ihl := uint8(data[0]) & 0x0F

                    // Truncated IP info
                    if len(data) < int(ihl*4) {
                        continue
                    }

                    srcIP = data[12:16]
                    data = data[ihl*4:]
                } else {
                    // Truncated IP info
                    if len(data) < 40 {
                        continue
                    }

                    srcIP = data[8:24]

                    data = data[40:]
                }

                // Truncated TCP info
                if len(data) < 13 {
                    continue
                }

                dataOffset := (data[12] & 0xF0) >> 4

                // We need only packets with data inside
                // Check that the buffer is larger than the size of the TCP header
                if len(data) > int(dataOffset*4) {
                    if !bpfSupported {
                        destPort := binary.BigEndian.Uint16(data[2:4])
                        srcPort := binary.BigEndian.Uint16(data[0:2])

                        // log.Println(t.port, destPort, srcPort, packet)

                        if !(destPort == t.port || (t.trackResponse && srcPort == t.port)) {
                            continue
                        }
                    }

                    newBuf := make([]byte, len(data)+16)
                    copy(newBuf[:16], srcIP)
                    copy(newBuf[16:], data)

                    t.packetsChan <- newBuf
                }
            }
        }(d)
    }

    wg.Wait()
    t.readyCh <- true
}