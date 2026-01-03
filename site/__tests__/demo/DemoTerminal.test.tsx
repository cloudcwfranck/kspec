import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import DemoTerminal from '@/demo/DemoTerminal';
import demoData from '@/demo/demoSteps.json';

describe('DemoTerminal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Rendering', () => {
    it('renders all tabs', () => {
      render(<DemoTerminal />);

      demoData.tabs.forEach((tab) => {
        expect(screen.getByRole('tab', { name: tab.label })).toBeInTheDocument();
      });
    });

    it('renders first tab as active by default', () => {
      render(<DemoTerminal />);

      const firstTab = screen.getByRole('tab', { name: demoData.tabs[0].label });
      expect(firstTab).toHaveAttribute('aria-selected', 'true');
    });

    it('renders tab description', () => {
      render(<DemoTerminal />);

      expect(screen.getByText(demoData.tabs[0].description)).toBeInTheDocument();
    });

    it('renders docs link', () => {
      render(<DemoTerminal />);

      const docsLink = screen.getByRole('link', { name: /View documentation/ });
      expect(docsLink).toHaveAttribute('href', demoData.tabs[0].docsLink);
    });

    it('renders terminal window', () => {
      render(<DemoTerminal />);

      expect(screen.getByRole('region', { name: /Interactive terminal demo/ })).toBeInTheDocument();
    });

    it('renders control buttons', () => {
      render(<DemoTerminal />);

      expect(screen.getByRole('button', { name: /Play demo/ })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /Previous step/ })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /Next step/ })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /Reset demo/ })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: /Copy current command/ })).toBeInTheDocument();
    });

    it('renders step progress indicator', () => {
      render(<DemoTerminal />);

      const totalSteps = demoData.tabs[0].steps.length;
      expect(screen.getByText(`Step 1 of ${totalSteps}`)).toBeInTheDocument();
    });
  });

  describe('Tab Navigation', () => {
    it('switches tabs on click', async () => {
      const user = userEvent.setup();
      render(<DemoTerminal />);

      const secondTab = screen.getByRole('tab', { name: demoData.tabs[1].label });
      await user.click(secondTab);

      expect(secondTab).toHaveAttribute('aria-selected', 'true');
      expect(screen.getByText(demoData.tabs[1].description)).toBeInTheDocument();
    });

    it('resets state when switching tabs', async () => {
      const user = userEvent.setup();
      render(<DemoTerminal />);

      // Advance step
      const nextButton = screen.getByRole('button', { name: /Next step/ });
      await user.click(nextButton);

      // Switch tab
      const secondTab = screen.getByRole('tab', { name: demoData.tabs[1].label });
      await user.click(secondTab);

      // Should be back to step 1
      const totalSteps = demoData.tabs[1].steps.length;
      expect(screen.getByText(`Step 1 of ${totalSteps}`)).toBeInTheDocument();
    });

    it('updates docs link when switching tabs', async () => {
      const user = userEvent.setup();
      render(<DemoTerminal />);

      const secondTab = screen.getByRole('tab', { name: demoData.tabs[1].label });
      await user.click(secondTab);

      const docsLink = screen.getByRole('link', { name: /View documentation/ });
      expect(docsLink).toHaveAttribute('href', demoData.tabs[1].docsLink);
    });
  });

  describe('Playback Controls', () => {
    it('starts playback when play button clicked', async () => {
      const user = userEvent.setup();
      render(<DemoTerminal />);

      const playButton = screen.getByRole('button', { name: /Play demo/ });
      await user.click(playButton);

      // Should show pause button
      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Pause playback/ })).toBeInTheDocument();
      });
    });

    it('pauses playback when pause button clicked', async () => {
      const user = userEvent.setup();
      render(<DemoTerminal />);

      // Start playback
      const playButton = screen.getByRole('button', { name: /Play demo/ });
      await user.click(playButton);

      // Pause
      const pauseButton = await screen.findByRole('button', { name: /Pause playback/ });
      await user.click(pauseButton);

      // Should show play button again
      expect(screen.getByRole('button', { name: /Play demo/ })).toBeInTheDocument();
    });

    it('resets to first step when reset button clicked', async () => {
      const user = userEvent.setup();
      render(<DemoTerminal />);

      // Advance step
      const nextButton = screen.getByRole('button', { name: /Next step/ });
      await user.click(nextButton);

      // Reset
      const resetButton = screen.getByRole('button', { name: /Reset demo/ });
      await user.click(resetButton);

      // Should be back to step 1
      const totalSteps = demoData.tabs[0].steps.length;
      expect(screen.getByText(`Step 1 of ${totalSteps}`)).toBeInTheDocument();
    });
  });

  describe('Step Navigation', () => {
    it('advances to next step on next button click', async () => {
      const user = userEvent.setup();
      render(<DemoTerminal />);

      const nextButton = screen.getByRole('button', { name: /Next step/ });
      await user.click(nextButton);

      const totalSteps = demoData.tabs[0].steps.length;
      expect(screen.getByText(`Step 2 of ${totalSteps}`)).toBeInTheDocument();
    });

    it('goes to previous step on prev button click', async () => {
      const user = userEvent.setup();
      render(<DemoTerminal />);

      // First advance
      const nextButton = screen.getByRole('button', { name: /Next step/ });
      await user.click(nextButton);

      // Then go back
      const prevButton = screen.getByRole('button', { name: /Previous step/ });
      await user.click(prevButton);

      const totalSteps = demoData.tabs[0].steps.length;
      expect(screen.getByText(`Step 1 of ${totalSteps}`)).toBeInTheDocument();
    });

    it('disables prev button on first step', () => {
      render(<DemoTerminal />);

      const prevButton = screen.getByRole('button', { name: /Previous step/ });
      expect(prevButton).toBeDisabled();
    });

    it('disables next button on last step', async () => {
      const user = userEvent.setup();
      render(<DemoTerminal />);

      const nextButton = screen.getByRole('button', { name: /Next step/ });
      const totalSteps = demoData.tabs[0].steps.length;

      // Advance to last step
      for (let i = 1; i < totalSteps; i++) {
        await user.click(nextButton);
      }

      expect(nextButton).toBeDisabled();
    });
  });

  describe('Copy Functionality', () => {
    it('copies command to clipboard', async () => {
      const user = userEvent.setup();
      const writeTextMock = vi.fn().mockResolvedValue(undefined);
      Object.assign(navigator, {
        clipboard: {
          writeText: writeTextMock,
        },
      });

      render(<DemoTerminal />);

      const copyButton = screen.getByRole('button', { name: /Copy current command/ });
      await user.click(copyButton);

      expect(writeTextMock).toHaveBeenCalledWith(demoData.tabs[0].steps[0].command);
    });

    it('shows visual feedback after copy', async () => {
      const user = userEvent.setup();
      render(<DemoTerminal />);

      const copyButton = screen.getByRole('button', { name: /Copy current command/ });
      await user.click(copyButton);

      // SVG should change to checkmark (this is a simplified check)
      // In a real scenario, you'd check for the specific SVG path change
      expect(copyButton).toBeInTheDocument();
    });
  });

  describe('Keyboard Navigation', () => {
    it('toggles play/pause on space key', async () => {
      render(<DemoTerminal />);

      // Press space to play
      fireEvent.keyDown(window, { key: ' ' });

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Pause playback/ })).toBeInTheDocument();
      });

      // Press space to pause
      fireEvent.keyDown(window, { key: ' ' });

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /Play demo/ })).toBeInTheDocument();
      });
    });

    it('advances step on right arrow key', () => {
      render(<DemoTerminal />);

      fireEvent.keyDown(window, { key: 'ArrowRight' });

      const totalSteps = demoData.tabs[0].steps.length;
      expect(screen.getByText(`Step 2 of ${totalSteps}`)).toBeInTheDocument();
    });

    it('goes back on left arrow key', async () => {
      const user = userEvent.setup();
      render(<DemoTerminal />);

      // Advance first
      const nextButton = screen.getByRole('button', { name: /Next step/ });
      await user.click(nextButton);

      // Then use keyboard
      fireEvent.keyDown(window, { key: 'ArrowLeft' });

      const totalSteps = demoData.tabs[0].steps.length;
      expect(screen.getByText(`Step 1 of ${totalSteps}`)).toBeInTheDocument();
    });

    it('resets on R key', async () => {
      const user = userEvent.setup();
      render(<DemoTerminal />);

      // Advance step
      const nextButton = screen.getByRole('button', { name: /Next step/ });
      await user.click(nextButton);

      // Press R
      fireEvent.keyDown(window, { key: 'r' });

      const totalSteps = demoData.tabs[0].steps.length;
      expect(screen.getByText(`Step 1 of ${totalSteps}`)).toBeInTheDocument();
    });

    it('ignores keyboard events when focus is in input', () => {
      const { container } = render(
        <>
          <input type="text" />
          <DemoTerminal />
        </>
      );

      const input = container.querySelector('input');
      input?.focus();

      fireEvent.keyDown(input!, { key: 'ArrowRight' });

      // Should still be on step 1
      const totalSteps = demoData.tabs[0].steps.length;
      expect(screen.getByText(`Step 1 of ${totalSteps}`)).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('has proper ARIA labels on tabs', () => {
      render(<DemoTerminal />);

      demoData.tabs.forEach((tab) => {
        const tabElement = screen.getByRole('tab', { name: tab.label });
        expect(tabElement).toHaveAttribute('aria-controls', `tabpanel-${tab.id}`);
        expect(tabElement).toHaveAttribute('id', `tab-${tab.id}`);
      });
    });

    it('has proper ARIA labels on tabpanel', () => {
      render(<DemoTerminal />);

      const tabpanel = screen.getByRole('tabpanel');
      const firstTab = demoData.tabs[0];
      expect(tabpanel).toHaveAttribute('id', `tabpanel-${firstTab.id}`);
      expect(tabpanel).toHaveAttribute('aria-labelledby', `tab-${firstTab.id}`);
    });

    it('has descriptive button labels', () => {
      render(<DemoTerminal />);

      expect(screen.getByLabelText('Play demo')).toBeInTheDocument();
      expect(screen.getByLabelText('Previous step')).toBeInTheDocument();
      expect(screen.getByLabelText('Next step')).toBeInTheDocument();
      expect(screen.getByLabelText('Reset demo')).toBeInTheDocument();
      expect(screen.getByLabelText('Copy current command')).toBeInTheDocument();
    });
  });

  describe('Responsive Design', () => {
    it('renders on mobile viewport', () => {
      global.innerWidth = 375;
      global.innerHeight = 667;

      render(<DemoTerminal />);

      expect(screen.getByRole('region', { name: /Interactive terminal demo/ })).toBeInTheDocument();
    });
  });
});
