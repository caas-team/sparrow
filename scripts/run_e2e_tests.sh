root=$(pwd)
export SPARROW_BIN=$(realpath ./dist/sparrow_linux_amd64_v1/sparrow)

EXIT_CODE=0

for i in $(ls e2e); do
    echo "Running test e2e/$i"
    cd e2e/$i
    ./test.sh || {
        EXIT_CODE=$?
        echo "E2E test e2e/$i failed"
    }
    cd $root
done

exit $EXIT_CODE