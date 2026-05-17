const frequencies = [440, 523, 659, 784];

let ctx: AudioContext | null = null;

function getCtx(): AudioContext {
  if (!ctx) {
    ctx = new AudioContext();
  }
  return ctx;
}

export async function playAlertSound(soundId: number): Promise<void> {
  const audioCtx = getCtx();
  if (audioCtx.state === 'suspended') {
    await audioCtx.resume();
  }
  const idx = Math.max(1, Math.min(4, soundId)) - 1;
  const osc = audioCtx.createOscillator();
  const gain = audioCtx.createGain();
  osc.type = 'sine';
  osc.frequency.value = frequencies[idx];
  gain.gain.setValueAtTime(0.2, audioCtx.currentTime);
  gain.gain.exponentialRampToValueAtTime(0.001, audioCtx.currentTime + 0.5);
  osc.connect(gain);
  gain.connect(audioCtx.destination);
  osc.start();
  osc.stop(audioCtx.currentTime + 0.5);
}
