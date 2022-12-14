from core.plot.plot_alignment import plot_alignment
from core.alignment.alignment import AlignmentForPlot, \
    AlignedNucleotideSequence, AlignedProteinSequence
from core.variables import nucleotide, protein, request_alignment_type, \
    request_alignment_text, encoding, request_alignment_format, request_show_similarity, \
        request_show_line_position, request_color_symbols
from .log import log

from typing import Dict
import io
import base64

def generate_figure_from_json(data: Dict) -> bytes:
    string = ''.join(b64_decode(data[request_alignment_text]))

    if data[request_alignment_type] == nucleotide:
        log.debug("Get nucleotide alignment")
        constructor = AlignedNucleotideSequence
    elif data[request_alignment_type] == protein:
        log.debug("Get protein alignment")
        constructor = AlignedProteinSequence
    else:
        raise ValueError(f"Not valid alignment type, possible values: {nucleotide}, {protein}")

    alignment = AlignmentForPlot(
        aln=io.StringIO(string),
        constructor=constructor,
        alignment_type=data[request_alignment_format].lower().replace("'", "")
        )

    return plot_alignment(
        alignment=alignment,
        show_line_position=data[request_show_line_position],
        color_symbols=data[request_color_symbols],
        show_similarity=data[request_show_similarity]
    )

def b64_decode(string: str) -> str:
    return base64.b64decode(string).decode(encoding)
