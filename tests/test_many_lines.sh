for i in {0..5000}; do
    echo -n "This is line $i" >> ./output.txt
    for j in {0..100}; do
        echo -n "$j " >> output.txt
    done

    echo "" >> output.txt
done















































