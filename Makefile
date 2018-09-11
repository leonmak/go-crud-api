build:
	cd api; go build -o main; mv main ..;

clean:
	rm main