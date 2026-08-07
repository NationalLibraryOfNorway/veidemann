import {ComponentFixture, TestBed} from '@angular/core/testing';

import {provideCoreTesting} from '../../../../core/core.testing.module';
import {SelectorComponent} from './selector.component';

describe('SelectorComponent', () => {
  let fixture: ComponentFixture<SelectorComponent>;
  let component: SelectorComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SelectorComponent],
      providers: [...provideCoreTesting],
    }).compileComponents();
    fixture = TestBed.createComponent(SelectorComponent);
    component = fixture.componentInstance;
    component.writeValue([]);
    fixture.detectChanges();
  });

  it('does not expose the label emoji picker', () => {
    expect(fixture.nativeElement.querySelector('[data-testid="emoji-picker-button"]')).toBeNull();
  });
});
