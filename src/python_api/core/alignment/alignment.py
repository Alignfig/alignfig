from biotite.sequence.align import Alignment
from biotite.sequence import Sequence, LetterAlphabet, AlphabetError
from Bio import AlignIO
from Bio.Align import MultipleSeqAlignment

from core.variables import encoding
from core.api.log import log

import numpy as np
from typing import List, Any, Tuple

class AlignedNucleotideSequence(Sequence):
    alphabet_unamb = LetterAlphabet(["A","C","G","T","-"])
    alphabet_amb   = LetterAlphabet(
        ["A","C","G","T","R","Y","W","S",
         "M","K","H","B","V","D","N","-"]
    )
    def __init__(self, sequence=[]):
        try:
            self._alphabet = AlignedNucleotideSequence.alphabet_unamb
            _ = self._alphabet.encode_multiple(sequence)
        except AlphabetError:
            self._alphabet = AlignedNucleotideSequence.alphabet_amb
            _ = self._alphabet.encode_multiple(sequence)
        super().__init__(sequence)

    def get_alphabet(self):
        return self._alphabet


class AlignedProteinSequence(Sequence):
    alphabet = LetterAlphabet(["A","C","D","E","F","G","H","I","K","L",
                               "M","N","P","Q","R","S","T","V","W","Y",
                               "B","Z","X","*","-"])

    def get_alphabet(self):
        return AlignedProteinSequence.alphabet


class AlignmentForPlot:
    def __init__(self, aln: bytes, constructor: Sequence, alignment_type: str):
        aln_list, headers = self.read_alignment_from_bytes(
            aln,
            constructor,
            alignment_type
            )

        log.debug('Generating traces from alignment')
        trace: np.ndarray = Alignment.trace_from_strings(aln_list)
        self.alignment = Alignment(sequences=aln_list, trace=trace)
        self.headers = headers

    def __repr__(self):
        return self._alignment.__repr__()

    def __str__(self):
        return self._alignment.__str__()

    def __len__(self):
        return len(self._alignment)

    def __iter__(self):
        return iter(self._alignment.sequences)

    def __getitem__(self, index):
        return self._alignment.sequences[index]

    @property
    def alignment(self):
        return self._alignment

    @alignment.setter
    def alignment(self, value):
        if not isinstance(value,  Alignment):
            raise ValueError
        self._alignment = value

    @property
    def headers(self):
        return self._headers

    @headers.setter
    def headers(self, value: Any):
        if not isinstance(value,  list):
            raise ValueError
        self._headers = value

    @property
    def sequences(self):
        return self._alignment.sequences
    @property
    def trace(self):
        return self._alignment.trace

    def get_gapped_sequences(self) -> List[str]:
        return self.alignment.get_gapped_sequences()

    @staticmethod
    def read_alignment_from_bytes(aln: bytes, constructor: Sequence, alignment_type: str) -> Tuple[List[str], List[str]]:
        aln_str_list: List[AlignedProteinSequence] = []
        aln_headers_list: List[str] = []

        log.debug('Reading alignment')
        aln_parser: MultipleSeqAlignment = AlignIO.read(aln, format=alignment_type)

        log.debug('Generating sequence list')
        for seq in aln_parser:
            aln_str_list.append(constructor(seq.seq))
            aln_headers_list.append(seq.id)
        return aln_str_list, aln_headers_list
