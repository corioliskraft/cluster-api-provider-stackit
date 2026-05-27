/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package util

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SetCondition adds or updates a condition on conditions.
//
// The LastTransitionTime is only refreshed when the status actually changes,
// so callers can call this unconditionally on every reconcile.
func SetCondition(conditions *[]metav1.Condition, condType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.Now()
	for i, c := range *conditions {
		if c.Type != condType {
			continue
		}
		if c.Status == status && c.Reason == reason && c.Message == message {
			return
		}
		(*conditions)[i].Status = status
		(*conditions)[i].Reason = reason
		(*conditions)[i].Message = message
		if c.Status != status {
			(*conditions)[i].LastTransitionTime = now
		}
		return
	}
	*conditions = append(*conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
	})
}
