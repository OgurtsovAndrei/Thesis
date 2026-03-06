import json
import os
import sys

# Setup path to import shared library
current_dir = os.path.dirname(os.path.abspath(__file__))
project_root = os.path.dirname(os.path.dirname(current_dir))
scripts_dir = os.path.join(project_root, "scripts")
sys.path.append(scripts_dir)

from bench_lib import plotter

def main():
    base_dir = os.path.dirname(os.path.abspath(__file__))
    input_file = os.path.join(base_dir, "raw", "p_length_distribution.json")
    output_dir = os.path.join(base_dir, "plots")
    
    plotter.ensure_dir(output_dir)
        
    with open(input_file, "r") as f:
        data = json.load(f)
        
    for item in data:
        L = item["L"]
        N = item["N"]
        PSize = item["P_size"]
        frequencies = item["frequencies"]
        
        # Convert string keys to float and frequencies to float for plotter
        # Plotter expects Dict[float, float]
        plot_data = {float(k): float(v) for k, v in frequencies.items()}
        
        title = f"String Length Distribution in Boundary Set P\nN={N}, L={L}, |P|={PSize} (|P|/N={PSize/N:.2f})"
        output_file = os.path.join(output_dir, f"p_length_distribution_L{L}.svg")
        
        plotter.draw_bar_chart(
            output_file,
            title,
            "Bit Length",
            "Frequency",
            plot_data
        )
        
        print(f"Generated plot: {output_file}")

if __name__ == "__main__":
    main()