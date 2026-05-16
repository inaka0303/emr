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
