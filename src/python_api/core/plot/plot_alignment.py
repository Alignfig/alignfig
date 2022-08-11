from core.alignment.alignment import AlignmentForPlot
from .common import savefig_to_b64
from .biotite_plot_func import plot_alignment_type_based
from core.api.log import log

import matplotlib.pyplot as plt


plt.switch_backend('Agg')

def plot_alignment(
    alignment: AlignmentForPlot, show_line_position=False,
    color_symbols=False, show_similarity=False,
) -> bytes:
    color_map = "-=#ffffff\nA=#7eff00\nC=#ffe300\nD=#ff0000\nE=#ff0000\nF=#7eff00\nG=#ff00e4\nH=#7eff00\nI=#7eff00\nK=#007bff\nL=#7eff00\nM=#7eff00\nN=#ff00e4\nP=#7eff00\nQ=#ff00e4\nR=#007bff\nS=#ff00e4\nT=#ff00e4\nV=#7eff00\nW=#7eff00\nX=#7eff00\nY=#ff00e4"
    color_map = {i.split('=')[0]: i.split('=')[1]
                 for i in color_map.split('\n')}

    log.debug('Creating figure and axis')
    fig, ax = plt.subplots(1, 1, figsize=(
        len(max(alignment, key=len)) / 3, len(alignment.sequences) / 2)
        )

    log.debug('Generating alignment picture')
    _ = plot_alignment_type_based(
            ax, alignment, labels=alignment.headers,
            symbols_per_line=len(alignment),
            label_size=15, symbol_size=12, number_size=15,
            similarity_kwargs={'func': similarity,
                               'label': 'Similarity',
                               'refseq': 0},
            show_similarity=show_similarity,
            show_line_position=show_line_position,
            color_symbols=color_symbols,
            color_scheme=list(map(color_map.get, alignment[0].get_alphabet()))
    )

    fig.tight_layout()
    return savefig_to_b64(fig)

def similarity(x: str, y: str):
    d = {
        '-': 0,
        'A': 1, 'F': 1, 'H': 1, 'I': 1, 'L': 1, 'M': 1,
        'P': 1, 'V': 1, 'W': 1, 'X': 1,
        'C': 2, 'D': 3, 'E': 3,
        'G': 4, 'N': 4, 'Q': 4, 'S': 4, 'T': 4, 'Y': 4,
        'K': 5, 'R': 5
    }
    res = [1 for i, j in enumerate(x) if d[j] == d[y[i]]]
    return f'{round(sum(res) * 100 / len(x), 1)}%'
