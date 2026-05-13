#!/usr/bin/env python3
import os
import sys
import subprocess
import shutil
import argparse
import difflib
import re

def flatten_latex(file_path, base_dir):
    if not os.path.exists(file_path):
        print(f"Warning: File {file_path} not found.")
        return ""
    
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    def replace_input(match):
        inc_file = match.group(1)
        if not inc_file.endswith('.tex'):
            inc_file += '.tex'
        inc_path = os.path.join(base_dir, inc_file)
        return flatten_latex(inc_path, base_dir)

    # Handle \input{...} and \include{...}
    content = re.sub(r'\\(?:input|include)\{([^}]+)\}', replace_input, content)
    return content

def main():
    parser = argparse.ArgumentParser(description='Highlight changed paragraphs in LaTeX.')
    parser.add_argument('commit', help='Git commit hash')
    parser.add_argument('--main', default='practical-range-emptiness.tex', help='Main file')
    parser.add_argument('--output', default='diff_paragraphs.pdf', help='Output PDF')
    args = parser.parse_args()

    script_dir = os.path.dirname(os.path.abspath(__file__))
    os.chdir(script_dir)
    
    main_file = args.main
    commit = args.commit

    # Setup temp dirs
    tmp_dir = os.path.join(script_dir, "diff_tmp_simple")
    if os.path.exists(tmp_dir):
        shutil.rmtree(tmp_dir)
    os.makedirs(tmp_dir)

    # 1. Get current flat content
    current_flat = flatten_latex(os.path.join(script_dir, main_file), script_dir)

    # 2. Get old flat content from Git
    # Find git root
    git_root = subprocess.check_output(["git", "rev-parse", "--show-toplevel"], text=True).strip()
    rel_path = os.path.relpath(script_dir, git_root)
    
    old_repo_dir = os.path.join(tmp_dir, "old_repo")
    os.makedirs(old_repo_dir)
    subprocess.run(f"git archive {commit} {rel_path} | tar -x -C {old_repo_dir}", shell=True, check=True, cwd=git_root)
    
    old_base_dir = os.path.join(old_repo_dir, rel_path)
    old_flat = flatten_latex(os.path.join(old_base_dir, main_file), old_base_dir)

    # 3. Compare by paragraphs
    # Split into blocks, preserving double newlines as much as possible
    old_paras = old_flat.split('\n\n')
    new_paras = current_flat.split('\n\n')

    matcher = difflib.SequenceMatcher(None, old_paras, new_paras)
    
    marked_content = []
    
    # Preamble patch
    preamble_patch = r"""
\usepackage[pdftex,color]{changebar}
\cbcolor{yellow}
\makeatletter
\@ifundefined{cbwidth}{%
  \@ifundefined{changebarwidth}{}{\setlength{\changebarwidth}{4pt}}%
}{%
  \setlength{\cbwidth}{4pt}%
}
\makeatother
\setlength{\changebarsep}{10pt}
"""

    for tag, i1, i2, j1, j2 in matcher.get_opcodes():
        if tag == 'equal':
            # Paragraphs are the same
            for j in range(j1, j2):
                marked_content.append(new_paras[j])
        elif tag in ('replace', 'insert'):
            # Paragraphs changed or added
            for j in range(j1, j2):
                para = new_paras[j].strip()
                if not para or len(para) < 5: # Skip very short snippets or empty lines
                    marked_content.append(new_paras[j])
                    continue
                
                # Check if it's preamble or sectioning we shouldn't wrap with bars usually
                if any(cmd in para for cmd in ["\\documentclass", "\\usepackage", "\\chapter", "\\section", "\\subsection"]):
                    marked_content.append(new_paras[j])
                else:
                    marked_content.append("\\cbstart\n" + new_paras[j] + "\n\\cbend")
        elif tag == 'delete':
            pass

    final_tex_content = "\n\n".join(marked_content)
    
    # Insert preamble patch
    if "\\begin{document}" in final_tex_content:
        final_tex_content = final_tex_content.replace("\\begin{document}", preamble_patch + "\n\\begin{document}")
    else:
        final_tex_content = preamble_patch + final_tex_content

    diff_tex = os.path.join(tmp_dir, "diff.tex")
    with open(diff_tex, 'w', encoding='utf-8') as f:
        f.write(final_tex_content)

    # 4. Compile
    print("--- Compiling ---")
    build_env = os.path.join(tmp_dir, "build")
    if os.path.exists(build_env):
        shutil.rmtree(build_env)
    shutil.copytree(script_dir, build_env, ignore=shutil.ignore_patterns('diff_tmp*', '*.pdf', 'build'))
    shutil.copy(diff_tex, os.path.join(build_env, "diff.tex"))
    
    try:
        # Added -f to force PDF generation despite missing figures/refs
        subprocess.run(["latexmk", "-pdf", "-f", "-interaction=nonstopmode", "diff.tex"], cwd=build_env, check=False)
        
        output_pdf = os.path.join(build_env, "diff.pdf")
        if os.path.exists(output_pdf):
            shutil.copy(output_pdf, os.path.join(script_dir, args.output))
            print(f"Success! PDF saved to {args.output}")
        else:
            print("PDF was not generated. Check build/diff.log")
    except Exception as e:
        print(f"Compilation process failed. Error: {e}")

if __name__ == "__main__":
    main()
