# Distributed Counter System

A fault-tolerant distributed counter system implemented in Go. Setiap node menyimpan counter lokal dan menggunakan CRDT dengan gossip untuk menyebarkan state ke peer lain. Sistem ini menyediakan API HTTP untuk increment, pembacaan nilai global secara eventual consistent, dan dynamic node join.

## Deskripsi Singkat

Sistem menyediakan counter terdistribusi yang:
- Menerima permintaan increment pada setiap node
- Menggabungkan nilai dari semua node aktif
- Menangani konflik dengan CRDT
- Menyediakan registrasi node baru statis dan dinamis
- Memiliki health check dengan timeout untuk mendeteksi node gagal

## Arsitektur dan Alur Komunikasi

### Komponen Utama

- `counter/crdt.go`
  - Implementasi PN Counter CRDT untuk conflict-free merge
  - Menyimpan `increments` dan `decrements` per node
- `node/node.go`
  - Menyimpan daftar peer dan statusnya
  - Mengelola peer health dan last-seen
- `gossip/protocol.go`
  - Sinkronisasi state periodik ke semua peer
  - Menggunakan HTTP `POST /gossip`
- `health/checker.go`
  - Memeriksa status `GET /health` pada peer
  - Menandai peer gagal setelah berulang kali timeout
- `api/handlers.go`
  - Menyediakan REST API untuk operasi counter dan cluster

### Alur Komunikasi

1. Node menerima `POST /counter/increment`
2. Counter lokal diperbarui menggunakan CRDT
3. Node memicu gossip ke peer aktif
4. Peer yang menerima gossip menggabungkan state menggunakan merge CRDT
5. Saat `GET /counter/value`, node membaca state lokal yang sudah digabungkan
6. Jika peer aktif tersedia, node dapat mem-fetch state mereka untuk memperbarui nilai sebelum merespon

## Cara Menjalankan (Multi-node)

### Menggunakan File Konfigurasi

Buat file konfigurasi JSON untuk setiap node seperti `config-node1.json`.

```bash
# Node 1
./distributed-counter -config config-node1.json

# Node 2
./distributed-counter -config config-node2.json

# Node 3
./distributed-counter -config config-node3.json
```

### Menggunakan Argumen Baris Perintah

```bash
# Node pertama
./distributed-counter -id node1 -addr localhost:8080

# Node kedua bergabung ke node pertama
./distributed-counter -id node2 -addr localhost:8081 -peer localhost:8080

# Node ketiga bergabung ke node pertama
./distributed-counter -id node3 -addr localhost:8082 -peer localhost:8080
```

### Menambahkan Node Dinamis

Jika node baru ingin bergabung setelah cluster berjalan:

```bash
curl -X POST http://localhost:8080/cluster/join \
  -H "Content-Type: application/json" \
  -d '{"node_id": "node4", "address": "localhost:8083"}'
```

## Strategi Conflict Resolution

Sistem menggunakan strategi CRDT `PNCounter` (Positive-Negative Counter):

- Setiap node menyimpan jumlah increment dan decrement sendiri
- Nilai global dihitung sebagai `sum(increments) - sum(decrements)`
- Merge dilakukan dengan mengambil nilai maksimum untuk setiap node pada field `increments` dan `decrements`

### Mengapa PN Counter?

- CRDT cocok untuk sistem terdistribusi dengan update konkuren tanpa koordinasi global
- Memungkinkan `increment` diterima pada node mana pun
- Menjamin eventual consistency saat gossip menyebarkan state
- Tidak memerlukan leader atau consensus protocol yang kompleks

## Keterbatasan Sistem

- Konsistensi bersifat eventual, bukan strongly consistent
- Jika node mati sebelum gossip mengirim update, nilai increment bisa tertunda sampai recovery
- Tidak ada mekanisme log persistence; restart node menghilangkan state lokal kecuali ditambahkan persistence
- Partisi jaringan dapat menyebabkan node menunjukkan nilai berbeda sementara
- Join propagation sederhana dan tidak mendukung topologi ring atau full membership broadcast secara efisien

## Instalasi

### Prasyarat

- Go 1.21 atau lebih tinggi

### Build

```bash
go mod download
go build -o distributed-counter
```

## Konfigurasi

Sistem dapat dikonfigurasi melalui file JSON atau argumen CLI.

### Format File Konfigurasi

```json
{
  "node_id": "node1",
  "listen_address": "localhost:8080",
  "peers": ["localhost:8081", "localhost:8082"],
  "gossip_interval_ms": 1000,
  "health_check_interval_ms": 5000,
  "request_timeout_ms": 5000
}
```

### Parameter Konfigurasi

- `node_id`: Identifier unik node
- `listen_address`: Host:port HTTP server
- `peers`: Daftar alamat peer awal
- `gossip_interval_ms`: Interval gossip dalam milidetik
- `health_check_interval_ms`: Interval health check dalam milidetik
- `request_timeout_ms`: Timeout HTTP dalam milidetik

## Dokumentasi API

Lihat `docs/API.md` untuk dokumentasi endpoint lengkap, parameter, contoh request/response, dan error code.

**Endpoint:** `GET /peers`

**Response:**
```json
{
  "peers": [
    {
      "id": "peer-localhost:8081",
      "address": "localhost:8081",
      "active": true,
      "last_seen": "2024-01-15T10:30:00Z"
    },
    {
      "id": "peer-localhost:8082",
      "address": "localhost:8082",
      "active": true,
      "last_seen": "2024-01-15T10:30:00Z"
    }
  ],
  "count": 2
}
```

**Example:**
```bash
curl http://localhost:8080/peers
```

## Conflict Resolution

The system uses a **PN Counter CRDT** for conflict resolution:

1. **Separate Tracking**: Increments and decrements are tracked separately per node
2. **Merge Function**: When merging counters from different nodes, the maximum value for each node is taken
3. **Eventual Consistency**: Through gossip protocol, all nodes eventually converge to the same state
4. **No Conflicts**: Concurrent updates are handled without conflicts due to the CRDT properties

### Example

```
Node A: increments by 5 → Counter: {A: +5}
Node B: increments by 3 → Counter: {B: +3}

After gossip:
Node A: Counter: {A: +5, B: +3} → Value: 8
Node B: Counter: {A: +5, B: +3} → Value: 8
```

## Fault Tolerance

The system provides fault tolerance through:

1. **Health Checks**: Periodic health checks detect failed nodes
2. **Peer Status Tracking**: Nodes track last-seen timestamps and failure counts
3. **Graceful Degradation**: System continues operating with available nodes
4. **No Single Point of Failure**: Any node can handle requests independently
5. **Automatic Recovery**: Failed nodes can rejoin and sync state via gossip

### Failure Detection

- Nodes are marked as failed after 3 consecutive health check failures
- Failed nodes are excluded from gossip but remain in peer list
- Failed nodes can recover and rejoin the cluster

## Testing the System

### Manual Testing

1. **Start three nodes:**
   ```bash
   # Terminal 1
   ./distributed-counter -id node1 -addr localhost:8080
   
   # Terminal 2
   ./distributed-counter -id node2 -addr localhost:8081 -peer localhost:8080
   
   # Terminal 3
   ./distributed-counter -id node3 -addr localhost:8082 -peer localhost:8080
   ```

2. **Increment counter on different nodes:**
   ```bash
   curl -X POST http://localhost:8080/counter/increment -H "Content-Type: application/json" -d '{"delta": 5}'
   curl -X POST http://localhost:8081/counter/increment -H "Content-Type: application/json" -d '{"delta": 3}'
   curl -X POST http://localhost:8082/counter/increment -H "Content-Type: application/json" -d '{"delta": 2}'
   ```

3. **Check values on all nodes:**
   ```bash
   curl http://localhost:8080/counter/value
   curl http://localhost:8081/counter/value
   curl http://localhost:8082/counter/value
   ```
   
   After gossip synchronization (within a few seconds), all nodes should show the same value (10).

4. **Test fault tolerance:**
   - Stop one node (Ctrl+C)
   - Continue incrementing on other nodes
   - Restart the stopped node
   - Verify it syncs via gossip

### Example Test Script

```bash
#!/bin/bash

# Increment on node1
curl -X POST http://localhost:8080/counter/increment \
  -H "Content-Type: application/json" \
  -d '{"delta": 1}'

# Wait for gossip
sleep 2

# Check values on all nodes
echo "Node 1: $(curl -s http://localhost:8080/counter/value | jq '.value')"
echo "Node 2: $(curl -s http://localhost:8081/counter/value | jq '.value')"
echo "Node 3: $(curl -s http://localhost:8082/counter/value | jq '.value')"
```

## Project Structure

```
.
├── main.go                 # Entry point
├── go.mod                  # Go module definition
├── README.md               # This file
├── config-node1.json       # Example config for node 1
├── config-node2.json       # Example config for node 2
├── config-node3.json       # Example config for node 3
├── counter/
│   └── crdt.go            # PN Counter CRDT implementation
├── node/
│   └── node.go            # Node and peer management
├── api/
│   └── handlers.go        # HTTP API handlers
├── gossip/
│   └── protocol.go        # Gossip protocol implementation
├── health/
│   └── checker.go         # Health check implementation
└── config/
    └── config.go          # Configuration management
```

## Limitations and Future Improvements

### Current Limitations

1. **No Authentication**: HTTP endpoints are not authenticated
2. **No Encryption**: Communication is not encrypted
3. **Memory Storage**: Counter state is not persisted to disk
4. **Basic Discovery**: No advanced service discovery mechanism
5. **Single Counter**: Only one counter per node (could be extended to multiple)

### Potential Improvements

1. **Persistence**: Add database or file-based persistence
2. **Security**: Add TLS and authentication
3. **Service Discovery**: Integrate with Consul, etcd, or similar
4. **Multiple Counters**: Support named counters
5. **Metrics**: Add Prometheus metrics export
6. **Websocket**: Real-time counter value updates
7. **Advanced Gossip**: Implement SWIM or similar gossip protocols
8. **Dynamic Scaling**: Auto-scaling and load balancing

## License

This is a educational project for a Distributed Systems course.

## Author

Implemented as a course project for Distributed Systems (Sistem Terdistribusi).

Ringkasan singkat, build dan jalankan (Windows)

Lokasi penting:
- `bin/` — berisi binary: `distributed-counter.exe`
- `scripts/windows/` — berisi helper scripts: `build.bat`, `start-nodes.bat`, `test.bat`

Contoh cepat (PowerShell dari root repo):

```powershell
.\n+cd "D:\Matkul Kuliah\Semester 6\Sistem Terdistribusi"
.
\scripts\windows\build.bat   # build binary ke bin\
\scripts\windows\start-nodes.bat  # buka 3 jendela node
\scripts\windows\test.bat   # jalankan test otomatis (menunggu readiness)
```

Catatan: semua skrip Windows otomatis berpindah ke root repo sehingga bisa dijalankan dari mana saja.
