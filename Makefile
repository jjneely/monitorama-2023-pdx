all:
	xelatex -shell-escape main.tex

clean:
	rm -f *.aux *.log *.nav *.out *.snm *.toc *.vrb
