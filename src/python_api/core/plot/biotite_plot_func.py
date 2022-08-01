from biotite.sequence.graphics import LetterTypePlotter
from typing import List


def plot_alignment_type_based(axes, alignment, symbols_per_line=50,
                              show_numbers=False, number_size=None,
                              number_functions=None,
                              labels=None, label_size=None,
                              show_line_position=False,
                              spacing=1,
                              color_scheme=None, color_symbols=False,
                              symbol_size=None, symbol_param=None,
                              symbol_spacing=None,
                              similarity_kwargs={}, show_similarity=False) -> List[float]:
    """
    Plot a pairwise or multiple sequence alignment coloring each symbol
    based on the symbol type.

    This function works like :func:`plot_alignment()` with a
    :class:`SymbolPlotter`, that colors the symbols based on a color
    scheme.
    The color intensity (or colormap value, respectively) of a symbol
    scales with similarity of the respective symbol to the other symbols
    in the same alignment column.

    Parameters
    ----------
    axes : Axes
        A *Matplotlib* axes, that is used as plotting area.
    alignment : Alignment
        The pairwise or multiple sequence alignment to be plotted.
        The alphabet of each sequence in the alignment must be the same.
    symbol_plotter : SymbolPlotter
        Defines how the symbols in the alignment are drawn.
    symbols_per_line : int, optional
        The amount of alignment columns that are diplayed per line.
    show_numbers : bool, optional
        If true, the sequence position of the symbols in the last
        alignment column of a line is shown on the right side of the
        plot.
        If the last symbol is a gap, the position of the last actual
        symbol before this gap is taken.
        If the first symbol did not occur up to this point,
        no number is shown for this line.
        By default the first symbol of a sequence has the position 1,
        but this behavior can be changed using the `number_functions`
        parameter.
    number_size : float, optional
        The font size of the position numbers
    number_functions : list of [(None or Callable(int -> int)], optional
        By default the position of the first symbol in a sequence is 1,
        i.e. the sequence position is the sequence index incremented by
        1.
        The behavior can be changed with this parameter:
        If supplied, the length of the list must match the number of
        sequences in the alignment.
        Every entry is a function that maps a sequence index (*int*) to
        a sequence position (*int*) for the respective sequence.
        A `None` entry means, that the default numbering is applied
        for the sequence.
    labels : list of str, optional
        The sequence labels.
        Must be the same size and order as the sequences in the
        alignment.
    label_size : float, optional
        Font size of the labels
    show_line_position : bool, optional
        If true the position within a line is plotted below the
        alignment.
    spacing : float, optional
        The spacing between the alignment lines. 1.0 means that the size
        is equal to the size of a symbol box.
    color_scheme : str or list of (tuple or str), optional
        Either a valid color scheme name
        (e.g. ``"rainbow"``, ``"clustalx"``, ``blossom``, etc.)
        or a list of *Matplotlib* compatible colors.
        The list length must be at least as long as the
        length of the alphabet used by the sequences.
    color_symbols : bool, optional
        If true, the symbols themselves are colored.
        If false, the symbols are black, and the boxes behind the
        symbols are colored.
    symbol_size : float, optional
        Font size of the sequence symbols.
    symbol_param : dict
        Additional parameters that is given to the
        :class:`matplotlib.Text` instance of each symbol.
    symbol_spacing : int, optional
        А space is placed between each number of elements desired
        by variable.
    show_similarity : bool, optional
        If True similarity_kwargs is used to plot similarity
        on the right side of the alignment. Can't be used with
        show_numbers equals to True.
    similarity_kwargs : dict, optional
        If show similarity is true is used for plotting similarity.
        keys to provide:
            'func': Callable, function to count similarity
                    (interface - func(x: str, y: str) -> str),
            'label': str, Label for column name
            'refseq': str, reference sequence to count alignment
                      (label or index from alignment)
    """

    from matplotlib.transforms import Bbox

    alphabet = alignment.sequences[0].get_alphabet()
    symbol_plotter = LetterTypePlotter(
        axes, alphabet, font_size=symbol_size, font_param=symbol_param,
        color_symbols=color_symbols, color_scheme=color_scheme
    )

    if number_functions is None:
        number_functions = [lambda x: x + 1] * len(alignment.sequences)
    else:
        if len(number_functions) != len(alignment.sequences):
            raise ValueError(
                f"The amount of renumbering functions is "
                f"{len(number_functions)} but the amount if sequences in the "
                f"alignment is {len(alignment.sequences)}"
            )
        for i, func in enumerate(number_functions):
            if func is None:
                number_functions[i] = (lambda x: x + 1)

    seq_num = alignment.trace.shape[1]
    seq_len = alignment.trace.shape[0]
    line_count = seq_len // symbols_per_line
    # Only extend line count by 1 if there is a remainder
    # (remaining symbols)
    if seq_len % symbols_per_line != 0:
        line_count += 1

    if symbol_spacing:
        spacing_ratio = symbols_per_line / symbol_spacing
        if spacing_ratio % 1 != 0:
            raise ValueError("symbols_per_line not multiple of symbol_spacing")
        # Initializing symbols_to_print to print symbols_per_line
        # symbols on one line + spacing between symbols
        symbols_to_print = int(spacing_ratio) + symbols_per_line - 1
    else:
        symbols_to_print = symbols_per_line

    ### Draw symbols ###
    x = 0
    y = 0
    y_start = 0
    line_pos = 0
    loops_traces = []
    for i in range(seq_len):
        y = y_start
        for j in range(seq_num):
            bbox = Bbox([[x, y], [x+1, y+1]])
            symbol_plotter.plot_symbol(bbox, alignment, i, j)

            y += 1

        loops_traces.append(bbox.p0[0] + bbox.width / 2)

        line_pos += 1
        if line_pos >= symbols_to_print:
            line_pos = 0
            x = 0
            y_start += seq_num + spacing
        else:
            x += 1
            if (symbol_spacing
               and (i + 1) % symbol_spacing == 0):
                line_pos += 1
                x += 1

    ### Draw labels ###
    ticks = []
    tick_labels = []
    if labels is not None:
        # Labels at center height of each line of symbols -> 0.5
        y = 0.5
        for i in range(line_count):
            for j in range(seq_num):
                ticks.append(y)
                tick_labels.append(labels[j])
                y += 1
            y += spacing
    axes.set_yticks(ticks)
    axes.set_yticklabels(tick_labels)

    ### Draw numbers ###
    # Create twin to allow different tick labels on right side
    number_axes = axes.twinx()
    ticks = []
    tick_labels = []

    ### Draw similarity ###
    if show_similarity:
        ticks.append(-0.6)
        tick_labels.append(similarity_kwargs['label'])
        y = 0.5
        ind = similarity_kwargs['refseq']
        ind = ind if isinstance(ind, int) else labels.index(ind)
        refseq = alignment.get_gapped_sequences()[ind]
        for i in range(line_count):
            for j in range(seq_num):
                ticks.append(y)
                tick_labels.append(similarity_kwargs['func'](
                    refseq,
                    alignment.get_gapped_sequences()[j]
                    ))
                y += 1
            y += spacing

    ### Draw numbers  ###
    if show_numbers:
        # Numbers at center height of each line of symbols -> 0.5
        y = 0.5
        for i in range(line_count):
            for j in range(seq_num):
                if i == line_count-1:
                    # Last line -> get number of last column in trace
                    trace_pos = len(alignment.trace) - 1
                else:
                    trace_pos = (i+1) * symbols_per_line - 1
                seq_index = _get_last_valid_index(
                    alignment, trace_pos, j
                )
                # if -1 -> terminal gap
                # -> skip number for this sequence in this line
                if seq_index != -1:
                    # Convert sequence index to position
                    # (default index + 1)
                    number = number_functions[j](seq_index)
                    ticks.append(y)
                    tick_labels.append(str(number))
                y += 1
            y += spacing
    number_axes.set_yticks(ticks)
    number_axes.set_yticklabels(tick_labels)

    axes.set_xlim(0, symbols_to_print)
    # Y-axis starts from top
    lim = seq_num*line_count + spacing*(line_count-1)
    axes.set_ylim(lim, 0)
    number_axes.set_ylim(lim, 0)
    axes.set_frame_on(False)
    number_axes.set_frame_on(False)
    # Remove ticks and set label and number size
    axes.yaxis.set_tick_params(
        left=False, right=False, labelsize=label_size
    )
    number_axes.yaxis.set_tick_params(
        left=False, right=False, labelsize=number_size
    )

    if show_line_position:
        axes.xaxis.set_tick_params(
            top=False, bottom=True, labeltop=False, labelbottom=True
        )
    else:
        axes.xaxis.set_tick_params(
            top=False, bottom=False, labeltop=False, labelbottom=False
        )
    return loops_traces

def _get_last_valid_index(alignment, column_i, seq_i):
    """
    Find the last trace value that belongs to a valid sequence index
    (no gap -> no -1) up to the specified column.
    """
    index_found = False
    while not index_found:
        if column_i == -1:
            # Iterated from column_i back to beyond the beginning
            # and no index has been found
            # -> Terminal gap
            # -> First symbol of sequence has not occured yet
            # -> return -1
            index = -1
            index_found = True
        else:
            index = alignment.trace[column_i, seq_i]
            if index != -1:
                index_found = True
        column_i -= 1
    return index
