package browser

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"flowpilot/internal/models"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

func (r *Runner) evaluateCondition(ctx context.Context, step models.TaskStep, vars map[string]string) (bool, error) {
	switch step.Action {
	case models.ActionIfElement:
		var nodes []*cdp.Node
		err := r.exec.Run(ctx, chromedp.Nodes(step.Selector, &nodes, chromedp.ByQuery, chromedp.AtLeast(0)))
		if err != nil {
			return false, fmt.Errorf("if_element check: %w", err)
		}
		switch step.Condition {
		case "not_exists":
			return len(nodes) == 0, nil
		default:
			return len(nodes) > 0, nil
		}

	case models.ActionIfText:
		var text string
		if err := r.exec.Run(ctx,
			chromedp.Text(step.Selector, &text, chromedp.ByQuery),
		); err != nil {
			return false, nil
		}
		return evaluateTextCondition(step.Condition, text, vars)

	case models.ActionIfURL:
		var currentURL string
		if err := r.exec.Run(ctx, chromedp.Location(&currentURL)); err != nil {
			return false, fmt.Errorf("if_url get location: %w", err)
		}
		return evaluateTextCondition(step.Condition, currentURL, vars)

	default:
		return false, fmt.Errorf("unknown condition action: %s", step.Action)
	}
}

func evaluateTextCondition(condition, text string, vars map[string]string) (bool, error) {
	for k, v := range vars {
		condition = strings.ReplaceAll(condition, "{{"+k+"}}", v)
	}

	parts := strings.SplitN(condition, ":", 2)
	if len(parts) != 2 {
		return strings.Contains(text, condition), nil
	}

	op, val := parts[0], parts[1]
	switch op {
	case "contains":
		return strings.Contains(text, val), nil
	case "not_contains":
		return !strings.Contains(text, val), nil
	case "equals":
		return text == val, nil
	case "not_equals":
		return text != val, nil
	case "starts_with":
		return strings.HasPrefix(text, val), nil
	case "ends_with":
		return strings.HasSuffix(text, val), nil
	case "matches":
		re, err := regexp.Compile(val)
		if err != nil {
			return false, fmt.Errorf("invalid regex in condition: %w", err)
		}
		return re.MatchString(text), nil
	default:
		return false, fmt.Errorf("unknown condition operator: %s", op)
	}
}

func buildLabelIndex(steps []models.TaskStep) map[string]int {
	idx := make(map[string]int, len(steps))
	for i, s := range steps {
		if s.Label != "" {
			idx[s.Label] = i
		}
	}
	return idx
}

func findEndLoop(steps []models.TaskStep, loopPC int) int {
	depth := 0
	for i := loopPC; i < len(steps); i++ {
		if steps[i].Action == models.ActionLoop {
			depth++
		}
		if steps[i].Action == models.ActionEndLoop {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}
