## Data Resources
- https://github.com/guzfirdaus/Wilayah-Administrasi-Indonesia
- https://github.com/cahyadsn/wilayah
  
## Installations
- import database
- create .env
- go run main.go

## API

- `/info` : informasi jumlah provinsi, kabupaten/kota, kecamatan, dan desa
- `/provinsi` : list provinsi
- `/provinsi/{id}` : detail provinsi
- `/kota` : list kabupaten/kota
- `/provinsi/{id}/kabupaten` : list kabupaten/kota berdasarkan provinsi
- `/kota/{id}` : detail kabupaten/kota
- `/kecamatan` : list kecamatan
- `/kota/{id}/kecamatan` : list kecamatan berdasarkan kabupaten/kota
- `/kecamatan/{id}` : detail kecamatan
- `/desa` : list desa
- `/kecamatan/{id}/desa` : list desa berdasarkan kecamatan
- `/desa/{id}` : detail desa

## Pagination
By default it has 10 limit with 1st page offset.

add more query param to address this: `?page=1&limit=1000`