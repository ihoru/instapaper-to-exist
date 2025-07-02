module github.com/ihoru/existio_instapaper

go 1.24

require github.com/ihoru/existio_instapaper/existio_client v0.0.0
require github.com/ihoru/existio_instapaper/storage v0.0.0

require github.com/joho/godotenv v1.5.1

replace github.com/ihoru/existio_instapaper/existio_client => ./existio_client
replace github.com/ihoru/existio_instapaper/storage => ./storage
