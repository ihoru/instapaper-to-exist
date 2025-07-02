module github.com/ihoru/instapaper-to-exist

go 1.24

require (
    github.com/ihoru/instapaper-to-exist/existio_client v0.1.0
    github.com/ihoru/instapaper-to-exist/storage v0.1.0
    github.com/joho/godotenv v1.5.1
)

//replace github.com/ihoru/instapaper-to-exist/existio_client => ./existio_client
//replace github.com/ihoru/instapaper-to-exist/storage => ./storage
