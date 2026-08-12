import {Component} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MatMenuModule} from '@angular/material/menu';

import {provideMaterialAnimationsDisabled} from '../../../core/core.testing.module';
import {DetailOverflowComponent} from './detail-overflow.component';

@Component({
  selector: 'app-test-menu-items',
  template: `<button mat-menu-item class="projected-action">Open related record</button>`,
  imports: [MatMenuModule],
  standalone: true,
})
class TestMenuItemsComponent {}

@Component({
  template: `
    <app-detail-overflow label="Page actions">
      <app-test-menu-items></app-test-menu-items>
    </app-detail-overflow>
  `,
  imports: [DetailOverflowComponent, TestMenuItemsComponent],
  standalone: true,
})
class DetailOverflowHostComponent {}

describe('DetailOverflowComponent', () => {
  let fixture: ComponentFixture<DetailOverflowHostComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [DetailOverflowHostComponent],
      providers: [provideMaterialAnimationsDisabled()],
    }).compileComponents();
    fixture = TestBed.createComponent(DetailOverflowHostComponent);
    fixture.detectChanges();
  });

  it('opens one accessible overflow menu containing projected items', async () => {
    const trigger = fixture.nativeElement.querySelector('button[aria-label="Page actions"]') as HTMLButtonElement;
    expect(trigger).not.toBeNull();
    expect(fixture.nativeElement.querySelectorAll('button[aria-haspopup="menu"]').length).toBe(1);

    trigger.click();
    fixture.detectChanges();
    await fixture.whenStable();
    await new Promise(resolve => setTimeout(resolve));

    const action = document.querySelector('.projected-action') as HTMLButtonElement;
    expect(action.textContent).toContain('Open related record');
    expect(document.activeElement).toBe(action);

    action.dispatchEvent(new KeyboardEvent('keydown', {key: 'ArrowDown', bubbles: true}));
    expect(document.activeElement).toBe(action);
  });
});
