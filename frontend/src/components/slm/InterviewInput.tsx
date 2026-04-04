import { useState, useRef, useCallback, useEffect } from 'react';

interface InterviewInputProps {
  onRequestSOAP: (text: string) => void;
  onRequestFamilyHistory: (text: string) => void;
  onRequestSocialHistory: (text: string) => void;
  isLoadingSOAP: boolean;
  isLoadingSummary: boolean;
  patientName: string;
}

const DEMO_PRESETS = [
  {
    label: '頭痛の患者',
    text: '3日前から持続的な頭痛があります。特に右側のこめかみあたりが痛みます。ズキズキする拍動性の痛みで、光や音に敏感になっています。市販のロキソニンを飲みましたが、あまり効きませんでした。吐き気もあります。母親も偏頭痛持ちです。仕事はデスクワークで1日10時間以上パソコンに向かっています。',
  },
  {
    label: '糖尿病フォロー',
    text: '定期受診です。前回のHbA1cが7.2%で、食事療法を頑張っていますが、なかなか下がりません。最近少し体重が増えました（2kg程度）。足の痺れは変わりありません。運動は週に2回ウォーキングをしています。父親が2型糖尿病で、インスリン注射をしていました。お酒は週末にビール1本程度です。',
  },
  {
    label: '腹痛の患者',
    text: '昨日の夕食後から腹痛が続いています。みぞおちの辺りがキリキリと痛みます。食事を取ると痛みが強くなります。下痢はありませんが、少し吐き気がします。熱は37.2℃でした。1週間前に焼肉を食べに行きました。兄が胃潰瘍の既往があります。喫煙は1日10本を15年間続けています。',
  },
];

function Spinner() {
  return (
    <svg
      className="animate-spin -ml-1 mr-2 h-4 w-4"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
    >
      <circle
        className="opacity-25"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        strokeWidth="4"
      />
      <path
        className="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      />
    </svg>
  );
}

export default function InterviewInput({
  onRequestSOAP,
  onRequestFamilyHistory,
  onRequestSocialHistory,
  isLoadingSOAP,
  isLoadingSummary,
  patientName,
}: InterviewInputProps) {
  const [text, setText] = useState('');
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const isEmpty = text.trim().length === 0;

  // テキストエリア自動リサイズ
  const autoResize = useCallback(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = 'auto';
    el.style.height = `${Math.max(el.scrollHeight, 240)}px`;
  }, []);

  useEffect(() => {
    autoResize();
  }, [text, autoResize]);

  return (
    <div className="bg-white rounded-lg border border-gray-200 shadow-sm">
      <div className="px-4 py-3 border-b border-gray-100">
        <h3 className="text-base font-semibold text-gray-800">問診入力</h3>
        <p className="text-xs text-gray-500 mt-0.5">
          {patientName} - 問診内容を自由記述してください
        </p>
      </div>

      <div className="p-4">
        {/* サンプル入力プリセット */}
        <div className="mb-3">
          <span className="text-xs font-medium text-gray-500 mr-2">サンプル入力:</span>
          <div className="inline-flex flex-wrap gap-2 mt-1">
            {DEMO_PRESETS.map((preset) => (
              <button
                key={preset.label}
                onClick={() => {
                  setText(preset.text);
                  // テキストエリアをリサイズ
                  setTimeout(() => autoResize(), 0);
                }}
                className="px-3 py-1 text-xs font-medium text-primary-700 bg-primary-50 border border-primary-200
                  rounded-full hover:bg-primary-100 hover:border-primary-300 transition-colors
                  focus:outline-none focus:ring-2 focus:ring-primary-300 focus:ring-offset-1"
              >
                {preset.label}
              </button>
            ))}
          </div>
        </div>

        <label
          htmlFor="interview-text"
          className="block text-sm font-medium text-gray-700 mb-2"
        >
          問診内容
        </label>
        <textarea
          id="interview-text"
          ref={textareaRef}
          value={text}
          onChange={(e) => setText(e.target.value)}
          onInput={autoResize}
          rows={10}
          placeholder="患者の訴え、症状の経過、現病歴などを自由に記述してください...&#10;&#10;例: 3日前からの頭痛を主訴に来院。痛みは前頭部に持続的で、市販の鎮痛剤では改善せず。&#10;仕事中にPC作業を長時間行うことが多い。食欲は正常、睡眠は頭痛のため浅い。&#10;父親が高血圧で治療中。本人は喫煙歴なし、飲酒は週2回程度。"
          className="w-full px-4 py-3 text-sm border border-gray-200 rounded-lg resize-none
            focus:outline-none focus:ring-2 focus:ring-primary-300 focus:border-transparent
            placeholder:text-gray-300 transition-colors leading-relaxed"
        />

        <div className="mt-4 flex flex-wrap gap-3">
          <button
            onClick={() => onRequestSOAP(text)}
            disabled={isEmpty || isLoadingSOAP}
            className="inline-flex items-center px-5 py-2.5 bg-primary-600 text-white text-sm font-medium
              rounded-lg hover:bg-primary-700 transition-colors focus:outline-none focus:ring-2
              focus:ring-primary-500 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isLoadingSOAP && <Spinner />}
            SOAP提案を依頼
          </button>

          <button
            onClick={() => onRequestFamilyHistory(text)}
            disabled={isEmpty || isLoadingSummary}
            className="inline-flex items-center px-4 py-2.5 bg-white text-gray-700 text-sm font-medium
              border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors focus:outline-none
              focus:ring-2 focus:ring-primary-300 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isLoadingSummary && <Spinner />}
            家族歴を抽出
          </button>

          <button
            onClick={() => onRequestSocialHistory(text)}
            disabled={isEmpty || isLoadingSummary}
            className="inline-flex items-center px-4 py-2.5 bg-white text-gray-700 text-sm font-medium
              border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors focus:outline-none
              focus:ring-2 focus:ring-primary-300 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isLoadingSummary && <Spinner />}
            社会歴を抽出
          </button>
        </div>
      </div>
    </div>
  );
}
