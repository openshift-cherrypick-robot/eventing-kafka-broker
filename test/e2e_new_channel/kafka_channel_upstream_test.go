//go:build e2e && cloudevents
// +build e2e,cloudevents

/*
 * Copyright 2022 The Knative Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package e2e_new_channel

import (
	"testing"

	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/reconciler-test/pkg/manifest"

	"github.com/cloudevents/sdk-go/v2/binding"
	"knative.dev/eventing/test/rekt/features/channel"
	ch "knative.dev/eventing/test/rekt/resources/channel"
	chimpl "knative.dev/eventing/test/rekt/resources/channel_impl"
	"knative.dev/eventing/test/rekt/resources/subscription"
	"knative.dev/pkg/system"
	_ "knative.dev/pkg/system/testing"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"

	// needed for vendoring the test resource files
	_ "knative.dev/eventing/test/rekt/resources/eventlibrary/events"
)

// TestChannelConformance
func TestChannelConformance(t *testing.T) {
	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.Managed(t),
	)

	channelName := "mychannelimpl"

	// Install and wait for a Ready Channel.
	env.Prerequisite(ctx, t, channel.ImplGoesReady(channelName))

	env.TestSet(ctx, t, channel.ControlPlaneConformance(channelName))
	env.TestSet(ctx, t, channel.DataPlaneConformance(channelName))
}

// TestSmoke_Channel
func TestSmoke_Channel(t *testing.T) {
	t.Parallel()

	ctx, env := global.Environment()
	t.Cleanup(env.Finish)

	names := []string{
		"customname",
		"name-with-dash",
		"name1with2numbers3",
		"name63-01234567890123456789012345678901234567890123456789012345",
	}

	for _, name := range names {
		env.Test(ctx, t, channel.GoesReady(name))
	}
}

// TestSmoke_ChannelImpl
func TestSmoke_ChannelImpl(t *testing.T) {
	t.Parallel()

	ctx, env := global.Environment()
	t.Cleanup(env.Finish)

	names := []string{
		"customname",
		"name-with-dash",
		"name1with2numbers3",
		"name63-01234567890123456789012345678901234567890123456789012345",
	}

	for _, name := range names {
		env.Test(ctx, t, channel.ImplGoesReady(name))
	}

}

// TestSmoke_ChannelWithSubscription
func TestSmoke_ChannelWithSubscription(t *testing.T) {
	t.Parallel()

	ctx, env := global.Environment()
	t.Cleanup(env.Finish)

	channelName := "mychannel"

	// Install and wait for a Ready Channel.
	env.Prerequisite(ctx, t, channel.GoesReady(channelName))
	chRef := ch.AsRef(channelName)

	names := []string{
		"customname",
		"name-with-dash",
		"name1with2numbers3",
		"name63-01234567890123456789012345678901234567890123456789012345",
	}

	for _, name := range names {
		env.Test(ctx, t, channel.SubscriptionGoesReady(name,
			subscription.WithChannel(chRef),
			subscription.WithSubscriber(nil, "http://example.com")),
		)
	}
}

// TestSmoke_ChannelImplWithSubscription
func TestSmoke_ChannelImplWithSubscription(t *testing.T) {
	t.Parallel()

	ctx, env := global.Environment()
	t.Cleanup(env.Finish)

	channelName := "mychannelimpl"

	// Install and wait for a Ready Channel.
	env.Prerequisite(ctx, t, channel.ImplGoesReady(channelName))
	chRef := chimpl.AsRef(channelName)

	names := []string{
		"customname",
		"name-with-dash",
		"name1with2numbers3",
		"name63-01234567890123456789012345678901234567890123456789012345",
	}

	for _, name := range names {
		env.Test(ctx, t, channel.SubscriptionGoesReady(name,
			subscription.WithChannel(chRef),
			subscription.WithSubscriber(nil, "http://example.com")),
		)
	}
}

/*
TestChannelChainByUsingReplyAsSubscriber tests the following scenario:

EventSource ---> (Channel ---> Subscription) x 10 ---> Sink

It uses Subscription's spec.reply as spec.subscriber.

This test should fail with https://github.com/knative/eventing/issues/5756 done.

*/
func TestChannelChainByUsingReplyAsSubscriber(t *testing.T) {
	// This test fails with our KafkaChannel here since the test assumes it is ok to leave Subscription's `spec.subscriber` empty.
	// Leaving that empty is not handled in the data channel as there won't be any `egress.Destination` in the contract resource.
	// Dataplane won't know what to do when the destination is empty.
	// See https://github.com/knative/eventing/issues/5756#issuecomment-1012996766
	t.Skip("We don't allow nil `spec.subscriber` in Subscription. See https://github.com/knative/eventing/issues/5756#issuecomment-1012996766")

	t.Parallel()

	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.Managed(t),
	)

	createSubscriberFn := func(ref *duckv1.KReference, uri string) manifest.CfgFn {
		return subscription.WithReply(ref, uri)
	}
	env.Test(ctx, t, channel.ChannelChain(10, createSubscriberFn))
}

/*
TestChannelChain tests the following scenario:

EventSource ---> (Channel ---> Subscription) x 10 ---> Sink

*/
func TestChannelChain(t *testing.T) {
	t.Parallel()

	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.Managed(t),
	)

	createSubscriberFn := func(ref *duckv1.KReference, uri string) manifest.CfgFn {
		return subscription.WithSubscriber(ref, uri)
	}
	env.Test(ctx, t, channel.ChannelChain(10, createSubscriberFn))
}

/*
TestChannelDeadLetterSink tests if the events that cannot be delivered end up in
the dead letter sink.

It uses Subscription's spec.reply as spec.subscriber.
*/
func TestChannelDeadLetterSink(t *testing.T) {
	t.Parallel()

	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.Managed(t),
	)

	createSubscriberFn := func(ref *duckv1.KReference, uri string) manifest.CfgFn {
		return subscription.WithSubscriber(ref, uri)
	}
	env.Test(ctx, t, channel.DeadLetterSink(createSubscriberFn))
}

/*
TestChannelDeadLetterSinkByUsingReplyAsSubscriber tests if the events that cannot be delivered end up in
the dead letter sink.

It uses Subscription's spec.reply as spec.subscriber.

This test should fail with https://github.com/knative/eventing/issues/5756 done.
*/
func TestChannelDeadLetterSinkByUsingReplyAsSubscriber(t *testing.T) {
	// This test fails with our KafkaChannel here since the test assumes it is ok to leave Subscription's `spec.subscriber` empty.
	// Leaving that empty is not handled in the data channel as there won't be any `egress.Destination` in the contract resource.
	// Dataplane won't know what to do when the destination is empty.
	// See https://github.com/knative/eventing/issues/5756#issuecomment-1012184663
	t.Skip("We don't allow nil `spec.subscriber` in Subscription. See https://github.com/knative/eventing/issues/5756#issuecomment-1012184663")

	t.Parallel()

	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.Managed(t),
	)

	createSubscriberFn := func(ref *duckv1.KReference, uri string) manifest.CfgFn {
		return subscription.WithReply(ref, uri)
	}
	env.Test(ctx, t, channel.DeadLetterSink(createSubscriberFn))
}

/*
TestEventTransformationForSubscription tests the following scenario:

             1            2                 5            6                  7
EventSource ---> Channel ---> Subscription ---> Channel ---> Subscription ----> Service(Logger)
                                   |  ^
                                 3 |  | 4
                                   |  |
                                   |  ---------
                                   -----------> Service(Transformation)
*/
func TestEventTransformationForSubscriptionV1(t *testing.T) {
	t.Parallel()

	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.Managed(t),
	)

	env.Test(ctx, t, channel.EventTransformation())
}

/*
TestBinaryEventForChannel tests the following scenario:

EventSource (binary-encoded messages) ---> Channel ---> Subscription ---> Service(Logger)

*/
func TestBinaryEventForChannel(t *testing.T) {
	t.Parallel()

	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.Managed(t),
	)

	env.Test(ctx, t, channel.SingleEventWithEncoding(binding.EncodingBinary))
}

/*
TestStructuredEventForChannel tests the following scenario:

EventSource (structured-encoded messages) ---> Channel ---> Subscription ---> Service(Logger)

*/
func TestStructuredEventForChannel(t *testing.T) {
	t.Parallel()

	ctx, env := global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.Managed(t),
	)

	env.Test(ctx, t, channel.SingleEventWithEncoding(binding.EncodingStructured))
}
