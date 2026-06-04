"""A small Pipecat voice pipeline for extractor testing."""

from pipecat.pipeline.pipeline import Pipeline
from pipecat.frames.frames import Frame
from pipecat.processors.frame_processor import FrameProcessor


class TranscriptionFrame(Frame):
    """Recognized text emitted by the STT service."""


class TextFrame(Frame):
    """LLM output text tokens."""


class WebRTCInput(FrameProcessor):
    """Receives microphone audio from a WebRTC peer."""


class DeepgramSTT(FrameProcessor):
    """Streaming speech-to-text service."""


class OpenAILLM(FrameProcessor):
    """Streaming LLM service."""


class CartesiaTTS(FrameProcessor):
    """Streaming text-to-speech service."""


def build_pipeline():
    """Construct the voice pipeline."""
    transport_in = WebRTCInput()
    stt = DeepgramSTT()
    llm = OpenAILLM()
    tts = CartesiaTTS()
    pipeline = Pipeline([transport_in, stt, llm, tts])
    return pipeline
