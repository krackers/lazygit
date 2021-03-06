package gui

import (
	"fmt"

	"github.com/jesseduffield/gocui"
	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
)

// list panel functions

func (gui *Gui) getSelectedRemoteBranch() *commands.RemoteBranch {
	selectedLine := gui.State.Panels.RemoteBranches.SelectedLine
	if selectedLine == -1 || len(gui.State.RemoteBranches) == 0 {
		return nil
	}

	return gui.State.RemoteBranches[selectedLine]
}

func (gui *Gui) handleRemoteBranchSelect(g *gocui.Gui, v *gocui.View) error {
	if gui.popupPanelFocused() {
		return nil
	}

	gui.State.SplitMainPanel = false

	if _, err := gui.g.SetCurrentView(v.Name()); err != nil {
		return err
	}

	gui.getMainView().Title = "Remote Branch"

	remoteBranch := gui.getSelectedRemoteBranch()
	if remoteBranch == nil {
		return gui.newStringTask("main", "No branches for this remote")
	}

	v.FocusPoint(0, gui.State.Panels.RemoteBranches.SelectedLine)

	if gui.inDiffMode() {
		return gui.renderDiff()
	}

	cmd := gui.OSCommand.ExecutableFromString(
		gui.GitCommand.GetBranchGraphCmdStr(remoteBranch.FullName()),
	)
	if err := gui.newCmdTask("main", cmd); err != nil {
		gui.Log.Error(err)
	}

	return nil
}

func (gui *Gui) handleRemoteBranchesEscape(g *gocui.Gui, v *gocui.View) error {
	return gui.switchBranchesPanelContext("remotes")
}

func (gui *Gui) renderRemoteBranchesWithSelection() error {
	branchesView := gui.getBranchesView()

	gui.refreshSelectedLine(&gui.State.Panels.RemoteBranches.SelectedLine, len(gui.State.RemoteBranches))
	displayStrings := presentation.GetRemoteBranchListDisplayStrings(gui.State.RemoteBranches, gui.State.Diff.Ref)
	gui.renderDisplayStrings(branchesView, displayStrings)
	if gui.g.CurrentView() == branchesView && branchesView.Context == "remote-branches" {
		if err := gui.handleRemoteBranchSelect(gui.g, branchesView); err != nil {
			return err
		}
	}

	return nil
}

func (gui *Gui) handleCheckoutRemoteBranch(g *gocui.Gui, v *gocui.View) error {
	remoteBranch := gui.getSelectedRemoteBranch()
	if remoteBranch == nil {
		return nil
	}
	if err := gui.handleCheckoutRef(remoteBranch.FullName(), handleCheckoutRefOptions{}); err != nil {
		return err
	}
	return gui.switchBranchesPanelContext("local-branches")
}

func (gui *Gui) handleMergeRemoteBranch(g *gocui.Gui, v *gocui.View) error {
	selectedBranchName := gui.getSelectedRemoteBranch().Name
	return gui.mergeBranchIntoCheckedOutBranch(selectedBranchName)
}

func (gui *Gui) handleDeleteRemoteBranch(g *gocui.Gui, v *gocui.View) error {
	remoteBranch := gui.getSelectedRemoteBranch()
	if remoteBranch == nil {
		return nil
	}
	message := fmt.Sprintf("%s '%s/%s'?", gui.Tr.SLocalize("DeleteRemoteBranchMessage"), remoteBranch.RemoteName, remoteBranch.Name)
	return gui.createConfirmationPanel(g, v, true, gui.Tr.SLocalize("DeleteRemoteBranch"), message, func(*gocui.Gui, *gocui.View) error {
		return gui.WithWaitingStatus(gui.Tr.SLocalize("DeletingStatus"), func() error {
			if err := gui.GitCommand.DeleteRemoteBranch(remoteBranch.RemoteName, remoteBranch.Name); err != nil {
				return err
			}

			return gui.refreshSidePanels(refreshOptions{scope: []int{BRANCHES, REMOTES}})
		})
	}, nil)
}

func (gui *Gui) handleRebaseOntoRemoteBranch(g *gocui.Gui, v *gocui.View) error {
	selectedBranchName := gui.getSelectedRemoteBranch().Name
	return gui.handleRebaseOntoBranch(selectedBranchName)
}

func (gui *Gui) handleSetBranchUpstream(g *gocui.Gui, v *gocui.View) error {
	selectedBranch := gui.getSelectedRemoteBranch()
	checkedOutBranch := gui.getCheckedOutBranch()

	message := gui.Tr.TemplateLocalize(
		"SetUpstreamMessage",
		Teml{
			"checkedOut": checkedOutBranch.Name,
			"selected":   selectedBranch.FullName(),
		},
	)

	return gui.createConfirmationPanel(g, v, true, gui.Tr.SLocalize("SetUpstreamTitle"), message, func(*gocui.Gui, *gocui.View) error {
		if err := gui.GitCommand.SetBranchUpstream(selectedBranch.RemoteName, selectedBranch.Name, checkedOutBranch.Name); err != nil {
			return err
		}

		return gui.refreshSidePanels(refreshOptions{scope: []int{BRANCHES, REMOTES}})
	}, nil)
}

func (gui *Gui) handleCreateResetToRemoteBranchMenu(g *gocui.Gui, v *gocui.View) error {
	selectedBranch := gui.getSelectedRemoteBranch()
	if selectedBranch == nil {
		return nil
	}

	return gui.createResetMenu(fmt.Sprintf("%s/%s", selectedBranch.RemoteName, selectedBranch.Name))
}

func (gui *Gui) handleNewBranchOffRemote(g *gocui.Gui, v *gocui.View) error {
	branch := gui.getSelectedRemoteBranch()
	if branch == nil {
		return nil
	}
	message := gui.Tr.TemplateLocalize(
		"NewBranchNameBranchOff",
		Teml{
			"branchName": branch.FullName(),
		},
	)
	return gui.createPromptPanel(g, v, message, branch.FullName(), func(g *gocui.Gui, v *gocui.View) error {
		if err := gui.GitCommand.NewBranch(gui.trimmedContent(v), branch.FullName()); err != nil {
			return gui.surfaceError(err)
		}
		gui.State.Panels.Branches.SelectedLine = 0
		if err := gui.switchBranchesPanelContext("local-branches"); err != nil {
			return err
		}
		return gui.refreshSidePanels(refreshOptions{mode: ASYNC})
	})
}
