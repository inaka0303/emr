# Experiment References

`experiment_references.jsonl` is the fixed reference file for the 8-case human
experiment. The SOAP text is imported from the reviewed Google Doc, while
`key_facts` is hand-curated for metric scoring.

Workflow:

```bash
scripts/build_experiment_references.py --output /tmp/experiment_references.generated.jsonl
scripts/curate_experiment_references.py \
  --input /tmp/experiment_references.generated.jsonl \
  --output references/experiment_references.jsonl
scripts/validate_experiment_references.py references/experiment_references.jsonl
```

Accepted specialist comments:

- `C1 / JC-AMI-A`: use clopidogrel instead of ticagrelor for P2Y12 wording;
  move amlodipine from continuation to acute hold/stop.

Manual LLM judge workflow:

```bash
scripts/export_current_experiment.sh
```

This writes `exports/judge_prompts_current.jsonl`. Each line contains one saved
attempt and a `prompt` field. Paste the prompt into GPT-5.5 / ChatGPT / Codex
and save the returned JSON objects as one-line JSONL:

```text
exports/judge_results_current.jsonl
```

Then merge the 5-axis judge scores into the score workbook:

```bash
scripts/merge_judge_results.py
```

The merged workbook is written to
`exports/experiment_scores_with_judge_current.xlsx`.

Text metric environment:

```bash
python3 -m pip install --user -r requirements-scoring.txt
scripts/export_current_experiment.sh
```

`export_current_experiment.sh` runs `score_experiment_results.py` with
`--with-bertscore --strict-text-deps`. The score workbook contains a
`scoring_environment` sheet, and the same metadata is written to
`exports/scoring_environment_current.json`. This records the ROUGE tokenizer,
BERTScore model, device, Python version, and package versions used for the run.
