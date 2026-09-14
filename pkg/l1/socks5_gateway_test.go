package l1

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func TestSOCKS5GatewayEchoTunnel(t *testing.T) {
	// 1. Iniciar un servidor Echo de prueba local
	echoListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Error creando echo listener: %v", err)
	}
	defer echoListener.Close()
	echoAddr := echoListener.Addr().String()

	go func() {
		for {
			conn, err := echoListener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_, _ = io.Copy(c, c)
			}(conn)
		}
	}()

	// 2. Iniciar el Gateway SOCKS5 en puerto efímero
	gw := NewSOCKS5Gateway("127.0.0.1:0", nil)
	if err := gw.Start(); err != nil {
		t.Fatalf("Error iniciando SOCKS5Gateway: %v", err)
	}
	defer gw.Stop()

	// Obtener puerto asignado al listener
	socksAddr := gw.listener.Addr().String()

	// 3. Cliente SOCKS5 manual
	client, err := net.DialTimeout("tcp", socksAddr, 2*time.Second)
	if err != nil {
		t.Fatalf("Error conectando a SOCKS5: %v", err)
	}
	defer client.Close()

	// Handshake: [0x05, 0x01, 0x00]
	_, _ = client.Write([]byte{0x05, 0x01, 0x00})
	resp := make([]byte, 2)
	if _, err := io.ReadFull(client, resp); err != nil {
		t.Fatalf("Fallo leyendo respuesta de handshake: %v", err)
	}
	if resp[0] != 0x05 || resp[1] != 0x00 {
		t.Fatalf("Handshake rechazado: %v", resp)
	}

	// Solicitud CONNECT al echo server
	host, portStr, _ := net.SplitHostPort(echoAddr)
	portNum := 0
	fmt.Sscanf(portStr, "%d", &portNum)

	ip := net.ParseIP(host).To4()
	req := []byte{0x05, 0x01, 0x00, 0x01}
	req = append(req, ip...)
	req = append(req, byte(portNum>>8), byte(portNum&0xFF))

	_, _ = client.Write(req)

	reply := make([]byte, 10)
	if _, err := io.ReadFull(client, reply); err != nil {
		t.Fatalf("Fallo leyendo respuesta de CONNECT: %v", err)
	}
	if reply[1] != 0x00 {
		t.Fatalf("CONNECT falló con código: %d", reply[1])
	}

	// 4. Enviar datos a través del túnel SOCKS5
	testMsg := []byte("Hola Soberano SOCKS5 en ipvn7")
	_, _ = client.Write(testMsg)

	buf := make([]byte, len(testMsg))
	if _, err := io.ReadFull(client, buf); err != nil {
		t.Fatalf("Fallo leyendo echo a través de SOCKS5: %v", err)
	}
	if string(buf) != string(testMsg) {
		t.Fatalf("Echo no coincide: %s vs %s", string(buf), string(testMsg))
	}

	stats := gw.GetStats()
	if stats["total_connections"].(uint64) < 1 {
		t.Fatalf("Estadísticas de conexiones no registradas")
	}
}
