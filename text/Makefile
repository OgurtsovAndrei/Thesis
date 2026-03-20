# Makefile for Thesis and Proposal Compilation

MAIN = practical-range-emptiness
PROPOSAL = proposal
OUT_DIR = out
FIG_DIR = figures
THESIS_ROOT = ..
BENCH_ROOT = ../..
.PHONY: all clean watch force proposal thesis figures

# SVG → PDF mapping (source → figures/<name>.pdf)
SVG_MAP = \
	$(THESIS_ROOT)/emptiness/are_soda_hash/soda_hash_mechanism.svg:soda_hash_mechanism \
	$(THESIS_ROOT)/emptiness/are_soda_hash/tradeoff_uniform_L128.svg:soda_tradeoff_uniform \
	$(THESIS_ROOT)/emptiness/are_soda_hash/tradeoff_clustered_L128.svg:soda_tradeoff_clustered \
	$(THESIS_ROOT)/emptiness/are_trunc/phantom_comparison.svg:phantom_comparison \
	$(THESIS_ROOT)/emptiness/are_trunc/tradeoff_uniform_L128.svg:trunc_tradeoff_uniform \
	$(THESIS_ROOT)/emptiness/are_pgm/cdf_mapping.svg:cdf_mapping \
	$(THESIS_ROOT)/emptiness/are_pgm/cdf_redistribution.svg:cdf_redistribution \
	$(THESIS_ROOT)/emptiness/are_hybrid/gap_equidistant.svg:gap_equidistant \
	$(THESIS_ROOT)/emptiness/are_hybrid_scan/segmentation.svg:segmentation \
	$(THESIS_ROOT)/emptiness/are_hybrid_scan/universe_comparison.svg:universe_comparison \
	$(THESIS_ROOT)/emptiness/are_hybrid_scan/fpr_bpk_books_L65536.svg:fpr_bpk_books_L65536 \
	$(THESIS_ROOT)/emptiness/are_hybrid_scan/fpr_bpk_wiki_L65536.svg:fpr_bpk_wiki_L65536 \
	$(BENCH_ROOT)/bench_results/plots/distributions/hist_sosd_fb.svg:hist_sosd_fb \
	$(BENCH_ROOT)/bench_results/plots/distributions/hist_sosd_books.svg:hist_sosd_books \
	$(BENCH_ROOT)/bench_results/plots/distributions/hist_sosd_wiki.svg:hist_sosd_wiki \
	$(BENCH_ROOT)/bench_results/plots/distributions/hist_sosd_osm.svg:hist_sosd_osm

figures:
	mkdir -p $(FIG_DIR)
	@for entry in $(SVG_MAP); do \
		src=$${entry%%:*}; \
		name=$${entry##*:}; \
		if [ -f "$$src" ]; then \
			echo "Converting $$name.svg → $$name.pdf"; \
			rsvg-convert -f pdf "$$src" -o $(FIG_DIR)/$$name.pdf; \
		else \
			echo "SKIP (not found): $$src"; \
		fi; \
	done

all: thesis proposal

thesis: figures
	mkdir -p $(OUT_DIR)
	latexmk -pdf -interaction=nonstopmode -outdir=$(OUT_DIR) $(MAIN).tex
	cp $(OUT_DIR)/$(MAIN).pdf .

proposal:
	mkdir -p $(OUT_DIR)
	latexmk -pdf -interaction=nonstopmode -outdir=$(OUT_DIR) $(PROPOSAL).tex
	cp $(OUT_DIR)/$(PROPOSAL).pdf .

clean:
	latexmk -C -outdir=$(OUT_DIR)
	rm -rf $(OUT_DIR)
	rm -f $(MAIN).pdf $(PROPOSAL).pdf

watch:
	mkdir -p $(OUT_DIR)
	latexmk -pvc -pdf -interaction=nonstopmode -outdir=$(OUT_DIR) $(MAIN).tex

force: figures
	mkdir -p $(OUT_DIR)
	latexmk -pdf -g -interaction=nonstopmode -outdir=$(OUT_DIR) $(MAIN).tex
	cp $(OUT_DIR)/$(MAIN).pdf .
