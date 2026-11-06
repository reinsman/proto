use prost::Message;
use rein_proto::hub::v1::{ExecutionEvent, RunState, execution_event};

#[test]
fn decodes_the_same_golden_event_as_go() {
    let bytes = include_bytes!("../../../../fixtures/execution-event.bin");
    let event = ExecutionEvent::decode(bytes.as_slice()).unwrap();
    let attempt = event.attempt.as_ref().unwrap();
    assert_eq!(attempt.run_id, "run-1");
    assert_eq!(attempt.attempt_id, "attempt-1");
    assert_eq!(event.sequence, 1);
    match event.payload.as_ref().unwrap() {
        execution_event::Payload::State(state) => {
            assert_eq!(state.state, RunState::Running as i32);
        }
        _ => panic!("expected state event"),
    }
    assert_eq!(event.encode_to_vec(), bytes);
}
