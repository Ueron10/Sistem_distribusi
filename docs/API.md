# API Documentation

Dokumentasi API untuk Distributed Counter System. Semua endpoint menggunakan format JSON dan mengembalikan kode status HTTP yang sesuai.

## 1. POST /counter/increment

Menaikkan nilai counter lokal dan menyebarkan perubahan ke peer aktif.

### Request

- Method: `POST`
- URL: `/counter/increment`
- Header: `Content-Type: application/json`
- Body:
```json
{
  "delta": 5
}
```

### Response

- Status: `200 OK`
- Body:
```json
{
  "value": 5,
  "node_id": "node1",
  "success": true,
  "message": "Counter incremented successfully"
}
```

### Notes

- `delta` bersifat opsional; jika tidak diberikan, default 1
- Increment diterapkan lokal terlebih dahulu, lalu gossip dikirim ke peer

### Error

- `400 Bad Request` jika body tidak valid
- `405 Method Not Allowed` jika method bukan POST

## 2. GET /counter/value

Mengambil nilai counter global dari perspektif node ini.

### Request

- Method: `GET`
- URL: `/counter/value`

### Response

- Status: `200 OK`
- Body:
```json
{
  "value": 10,
  "node_id": "node1",
  "peer_count": 2,
  "active_peers": 2
}
```

### Notes

- Node akan mengambil state dari peer aktif terlebih dahulu untuk memperbarui hasil
- Nilai bersifat eventual consistent

### Error

- `405 Method Not Allowed` jika method bukan GET

## 3. GET /counter/state

Mengambil state CRDT mentah dari node ini.

### Request

- Method: `GET`
- URL: `/counter/state`

### Response

- Status: `200 OK`
- Body:
```json
{
  "node_id": "node1",
  "peer_count": 2,
  "active_peers": 2,
  "state": {
    "increments": {
      "node1": 5,
      "node2": 3
    },
    "decrements": {}
  }
}
```

### Notes

- Endpoint ini digunakan internal untuk agregasi state
- Mendukung debug dan inspeksi CRDT

### Error

- `405 Method Not Allowed` jika method bukan GET

## 4. POST /cluster/join

Mendaftarkan node baru ke cluster.

### Request

- Method: `POST`
- URL: `/cluster/join`
- Header: `Content-Type: application/json`
- Body:
```json
{
  "node_id": "node3",
  "address": "localhost:8082"
}
```

### Response

- Status: `200 OK`
- Body:
```json
{
  "success": true,
  "message": "Node joined cluster successfully",
  "node_id": "node3",
  "peer_count": 3
}
```

### Notes

- Node baru akan ditambahkan ke daftar peer lokal
- Join akan dipropagasi ke peer lain dalam cluster jika `propagate` tidak di-set ke false

### Error

- `400 Bad Request` jika `node_id` atau `address` kosong
- `400 Bad Request` jika `address` tidak valid
- `405 Method Not Allowed` jika method bukan POST

## 5. GET /health

Mengecek status kesehatan node.

### Request

- Method: `GET`
- URL: `/health`

### Response

- Status: `200 OK`
- Body:
```json
{
  "status": "healthy",
  "node_id": "node1",
  "uptime": "5m23s",
  "peer_count": 2,
  "active_peers": 2
}
```

### Notes

- Health check dipakai peer lain untuk mendeteksi node aktif

### Error

- `405 Method Not Allowed` jika method bukan GET

## 6. GET /peers

Melihat list peer yang diketahui.

### Request

- Method: `GET`
- URL: `/peers`

### Response

- Status: `200 OK`
- Body:
```json
{
  "peers": [
    {
      "id": "node2",
      "address": "localhost:8081",
      "active": true,
      "last_seen": "2026-05-27T12:30:45Z"
    }
  ],
  "count": 1
}
```

### Error

- `405 Method Not Allowed` jika method bukan GET
