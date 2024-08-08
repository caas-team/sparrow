root=$(pwd)
export SPARROW_BIN=$(realpath ./dist/sparrow_linux_amd64_v1/sparrow)

EXIT_CODE=0

MAX_RETRY=3

for i in $(ls e2e); do
    for ATTEMPT in $(seq 1 $MAX_RETRY ); do
        echo "[$ATTEMPT/$MAX_RETRY] Running test e2e/$i"
        cd e2e/$i
        ./test.sh 
        TEST_EXIT_CODE=$?
        cd $root
        if [ $TEST_EXIT_CODE -eq 0 ]; then
            break
        elif [ $ATTEMPT -eq $MAX_RETRY ]; then
            EXIT_CODE=1
        fi
    done 
done

exit $EXIT_CODE