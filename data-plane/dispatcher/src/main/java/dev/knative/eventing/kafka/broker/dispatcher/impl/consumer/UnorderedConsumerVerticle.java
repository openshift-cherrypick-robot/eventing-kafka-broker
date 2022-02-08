/*
 * Copyright © 2018 Knative Authors (knative-dev@googlegroups.com)
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
package dev.knative.eventing.kafka.broker.dispatcher.impl.consumer;

import dev.knative.eventing.kafka.broker.dispatcher.DeliveryOrder;
import io.cloudevents.CloudEvent;
import io.vertx.core.CompositeFuture;
import io.vertx.core.Future;
import io.vertx.core.Promise;
import io.vertx.kafka.client.consumer.KafkaConsumerRecords;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;
import java.util.ArrayList;
import java.util.Set;

import static dev.knative.eventing.kafka.broker.core.utils.Logging.keyValue;

/**
 * This {@link io.vertx.core.Verticle} implements an unordered consumer logic, as described in {@link DeliveryOrder#UNORDERED}.
 */
public final class UnorderedConsumerVerticle extends BaseConsumerVerticle {

  private static final Logger logger = LoggerFactory.getLogger(UnorderedConsumerVerticle.class);

  private static final long BACKOFF_DELAY_MS = 200;
  // This shouldn't be more than 2000, which is the default max time allowed
  // to block a verticle thread.
  private static final Duration POLL_TIMEOUT = Duration.ofMillis(1000);

  public UnorderedConsumerVerticle(Initializer initializer,
                                   Set<String> topics) {
    super(initializer, topics);
  }

  @Override
  void startConsumer(Promise<Void> startPromise) {
    this.consumer.exceptionHandler(this::exceptionHandler);
    this.consumer.subscribe(this.topics, startPromise);

    startPromise.future()
      .onSuccess(v -> poll());
  }

  /**
   * Vert.x auto-subscribe and handling of records might grow
   * unbounded, and it is particularly evident when the consumer
   * is slow to consume messages.
   * <p>
   * To apply backpressure, we need to bound the number of outbound
   * in-flight requests, so we need to manually poll for new records
   * as we dispatch them to the subscriber service.
   * <p>
   * The maximum number of outbound in-flight requests is already configurable
   * with the consumer parameter `max.poll.records`, and it's critical to
   * control the memory consumption of the dispatcher.
   */
  private void poll() {
    this.consumer
      .poll(POLL_TIMEOUT)
      .onSuccess(this::handleRecords)
      .onFailure(cause -> {
        logger.error("Failed to poll messages {}", keyValue("topics", topics), cause);
        // Wait before retrying.
        vertx.setTimer(BACKOFF_DELAY_MS, t -> poll());
      });
  }

  private void handleRecords(final KafkaConsumerRecords<Object, CloudEvent> records) {
    final var futures = new ArrayList<Future>(records.size());
    for (int i = 0; i < records.size(); i++) {
      futures.add(this.recordDispatcher.dispatch(records.recordAt(i)));
    }
    CompositeFuture.join(futures)
      .onComplete(v -> poll());
  }
}