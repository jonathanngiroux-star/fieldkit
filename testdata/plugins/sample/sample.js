// Fieldkit sample plugin (api_version 1, kind: question_type).
// A plugin is a JSON manifest plus an entrypoint file the host loads.
// The contract: the manifest parses, the entrypoint exists and is
// non-empty, and the host can render/validate the declared type.
// This sample is intentionally trivial — it exists so the contract
// check has a real subject in CI, not to showcase features.
(function () {
  return {
    type: "slider",
    render: function (el, question, answer) {
      el.textContent = question.title + " [" + (question.min || 0) + ".." + (question.max || 100) + "]";
    },
    validate: function (answer, question) {
      var n = Number(answer);
      if (isNaN(n)) return "must be a number";
      var lo = question.min || 0, hi = question.max || 100;
      if (n < lo || n > hi) return "must be between " + lo + " and " + hi;
      return null;
    }
  };
})();
