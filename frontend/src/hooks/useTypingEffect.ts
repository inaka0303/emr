import { useState, useEffect, useRef } from 'react';

interface UseTypingEffectResult {
  displayText: string;
  isTyping: boolean;
  /** タイピングを中断し、全文を即座に返す */
  complete: () => void;
}

export function useTypingEffect(
  text: string,
  speed: number = 25,
): UseTypingEffectResult {
  const [displayText, setDisplayText] = useState('');
  const [isTyping, setIsTyping] = useState(false);
  const indexRef = useRef(0);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const textRef = useRef(text);

  useEffect(() => {
    // テキストが変わったらタイピングをリセット
    textRef.current = text;
    indexRef.current = 0;
    setDisplayText('');

    if (!text) {
      setIsTyping(false);
      return;
    }

    setIsTyping(true);

    const tick = () => {
      if (indexRef.current < textRef.current.length) {
        indexRef.current += 1;
        setDisplayText(textRef.current.slice(0, indexRef.current));
        timerRef.current = setTimeout(tick, speed);
      } else {
        setIsTyping(false);
      }
    };

    timerRef.current = setTimeout(tick, speed);

    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
        timerRef.current = null;
      }
    };
  }, [text, speed]);

  const complete = () => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    setDisplayText(textRef.current);
    setIsTyping(false);
  };

  return { displayText, isTyping, complete };
}
